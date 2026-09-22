import UIKit

final class ReadingListViewController: BaseViewController {
  private enum Constants {
    static let pageLimit = 20
  }

  private enum LoadFailureState {
    case offline
    case generic

    var title: String {
      switch self {
      case .offline:
        "No internet connection"
      case .generic:
        "Couldn't load your Bookmarks."
      }
    }

    var message: String {
      switch self {
      case .offline:
        "We'll keep trying to reconnect."
      case .generic:
        "Try again in a moment."
      }
    }
  }

  private let environment: AppEnvironment
  private var posts: [Post] = []
  private var nextCursor: String?
  private var isLoadingPage = false
  private var inFlightLikePostIDs = Set<String>()
  private var loadFailureState: LoadFailureState?
  private var pendingRemovedBookmarkIndexes: [String: Int] = [:]
  private var needsTableReloadOnAppear = false

  private let tableView: UITableView = {
    let tableView = UITableView(frame: .zero, style: .plain)
    tableView.translatesAutoresizingMaskIntoConstraints = false
    tableView.separatorStyle = .none
    tableView.rowHeight = UITableView.automaticDimension
    tableView.estimatedRowHeight = 120
    return tableView
  }()

  private let refreshControl = UIRefreshControl()

  private let emptyStateTitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.bodyEditorialLarge
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.textAlignment = .center
    label.numberOfLines = 0
    label.text = "You have no Bookmarks."
    return label
  }()

  private let emptyStateSubtitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .center
    label.numberOfLines = 0
    label.text = "Save posts you want to read later."
    return label
  }()

  private let loadingIndicator: UIActivityIndicatorView = {
    let indicator = UIActivityIndicatorView(style: .medium)
    indicator.color = Theme.secondaryText
    indicator.hidesWhenStopped = true
    return indicator
  }()

  private lazy var backgroundStack: UIStackView = {
    let stack = UIStackView(arrangedSubviews: [loadingIndicator, emptyStateTitleLabel, emptyStateSubtitleLabel])
    stack.axis = .vertical
    stack.alignment = .center
    stack.spacing = 12
    stack.isLayoutMarginsRelativeArrangement = true
    stack.layoutMargins = UIEdgeInsets(top: 32, left: 24, bottom: 32, right: 24)
    return stack
  }()

  private let backgroundContainer = UIView()
  private let inlineStatusView = InlineStatusView()

  private let paginationSpinner = UIActivityIndicatorView(style: .medium)

  init(environment: AppEnvironment) {
    self.environment = environment
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func viewDidLoad() {
    super.viewDidLoad()
    configureNavigationTitle()
    setupTable()

    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleBookmarkStateChange(_:)),
      name: .bookmarkStateDidChange,
      object: nil
    )

    posts = environment.bookmarkStore.cachedReadingListPosts()
    updateBackgroundState()
    loadReadingList(reset: true)
  }

  override func viewDidAppear(_ animated: Bool) {
    super.viewDidAppear(animated)
    if needsTableReloadOnAppear {
      tableView.reloadData()
      needsTableReloadOnAppear = false
    }
  }

  private func configureNavigationTitle() {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.navigationTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = "Bookmarks"
    navigationItem.titleView = titleLabel
  }

  private func setupTable() {
    tableView.register(PostTableViewCell.self, forCellReuseIdentifier: PostTableViewCell.reuseID)
    tableView.dataSource = self
    tableView.delegate = self
    tableView.refreshControl = refreshControl
    refreshControl.addTarget(self, action: #selector(refreshTapped), for: .valueChanged)

    inlineStatusView.isHidden = true
    inlineStatusView.onActionTap = { [weak self] in
      self?.loadReadingList(reset: true)
    }
    backgroundContainer.addSubview(backgroundStack)
    backgroundContainer.addSubview(inlineStatusView)
    backgroundStack.translatesAutoresizingMaskIntoConstraints = false
    NSLayoutConstraint.activate([
      backgroundStack.centerYAnchor.constraint(equalTo: backgroundContainer.centerYAnchor),
      backgroundStack.leadingAnchor.constraint(equalTo: backgroundContainer.leadingAnchor),
      backgroundStack.trailingAnchor.constraint(equalTo: backgroundContainer.trailingAnchor),
      inlineStatusView.topAnchor.constraint(equalTo: backgroundContainer.topAnchor),
      inlineStatusView.bottomAnchor.constraint(equalTo: backgroundContainer.bottomAnchor),
      inlineStatusView.leadingAnchor.constraint(equalTo: backgroundContainer.leadingAnchor),
      inlineStatusView.trailingAnchor.constraint(equalTo: backgroundContainer.trailingAnchor)
    ])
    tableView.backgroundView = backgroundContainer

    paginationSpinner.color = Theme.secondaryText

    view.addSubview(tableView)
    NSLayoutConstraint.activate([
      tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor)
    ])
  }

  @objc private func refreshTapped() {
    loadReadingList(reset: true)
  }

  private func loadReadingList(reset: Bool) {
    guard !isLoadingPage else { return }
    if !reset, nextCursor == nil { return }

    isLoadingPage = true
    if reset {
      loadFailureState = nil
    }
    updateBackgroundState()
    updatePaginationState(isLoadingMore: !reset && !posts.isEmpty)
    let hadVisiblePosts = reset && !posts.isEmpty

    Task {
      defer {
        isLoadingPage = false
        refreshControl.endRefreshing()
        updateBackgroundState()
        updatePaginationState(isLoadingMore: false)
      }

      do {
        let page = try await environment.api.fetchBookmarksPage(
          limit: Constants.pageLimit,
          cursor: reset ? nil : nextCursor
        )
        let resolvedItems = environment.bookmarkStore.absorbReadingListPosts(page.items, reset: reset)

        if reset {
          posts = resolvedItems
          loadFailureState = nil
          if isInVisibleHierarchy {
            tableView.reloadData()
          } else {
            needsTableReloadOnAppear = true
          }
        } else {
          let start = posts.count
          let uniqueItems = resolvedItems.filter { incoming in
            !posts.contains(where: { $0.id == incoming.id })
          }
          posts.append(contentsOf: uniqueItems)
          if !uniqueItems.isEmpty {
            if isInVisibleHierarchy {
              let indexPaths = (start ..< posts.count).map { IndexPath(row: $0, section: 0) }
              tableView.performBatchUpdates({
                tableView.insertRows(at: indexPaths, with: .none)
              })
            } else {
              needsTableReloadOnAppear = true
            }
          }
        }

        nextCursor = page.nextCursor
      } catch {
        if posts.isEmpty && !hadVisiblePosts {
          loadFailureState = shouldPresentOfflineLoadState(for: error) ? .offline : .generic
        } else if reset || refreshControl.isRefreshing {
          Haptics.shared.error()
          environment.feedbackCenter.showRecoverableActionFailure(.refreshBookmarks)
        }
      }
    }
  }

  private func updateBackgroundState() {
    let isEmpty = posts.isEmpty
    tableView.backgroundView?.isHidden = !isEmpty
    guard isEmpty else {
      loadingIndicator.stopAnimating()
      backgroundStack.isHidden = true
      inlineStatusView.isHidden = true
      return
    }

    if let loadFailureState {
      loadingIndicator.stopAnimating()
      backgroundStack.isHidden = true
      inlineStatusView.isHidden = false
      inlineStatusView.configure(
        title: loadFailureState.title,
        message: loadFailureState.message,
        actionTitle: "Retry"
      )
      return
    }

    inlineStatusView.isHidden = true
    backgroundStack.isHidden = false
    emptyStateTitleLabel.isHidden = isLoadingPage
    emptyStateSubtitleLabel.isHidden = isLoadingPage

    if isLoadingPage {
      loadingIndicator.startAnimating()
    } else {
      loadingIndicator.stopAnimating()
    }
  }

  private func updatePaginationState(isLoadingMore: Bool) {
    guard isLoadingMore else {
      paginationSpinner.stopAnimating()
      tableView.tableFooterView = UIView(frame: .init(x: 0, y: 0, width: 1, height: 8))
      return
    }

    paginationSpinner.startAnimating()
    paginationSpinner.frame = CGRect(x: 0, y: 0, width: tableView.bounds.width, height: 56)
    tableView.tableFooterView = paginationSpinner
  }

  private func toggleLike(for postID: String) {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }
    guard !inFlightLikePostIDs.contains(postID) else { return }
    guard let index = posts.firstIndex(where: { $0.id == postID }) else { return }

    let original = posts[index]
    let desiredLikedState = !original.liked
    inFlightLikePostIDs.insert(postID)
    posts[index] = original.withLiked(desiredLikedState)
    refreshRow(at: index)

    Task {
      do {
        if desiredLikedState {
          try await environment.api.likePost(id: postID)
        } else {
          try await environment.api.unlikePost(id: postID)
        }
        inFlightLikePostIDs.remove(postID)
        guard let updatedIndex = posts.firstIndex(where: { $0.id == postID }) else { return }
        guard posts[updatedIndex].liked == desiredLikedState else { return }
        Haptics.shared.softImpact()
        let updatedIndexPath = IndexPath(row: updatedIndex, section: 0)
        if let cell = tableView.cellForRow(at: updatedIndexPath) as? PostTableViewCell {
          cell.animateLikeTransition()
        }
      } catch {
        inFlightLikePostIDs.remove(postID)
        guard let updatedIndex = posts.firstIndex(where: { $0.id == postID }) else { return }
        guard posts[updatedIndex].liked == desiredLikedState else { return }
        posts[updatedIndex] = original
        refreshRow(at: updatedIndex)
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(
          desiredLikedState ? .respect : .removeRespect
        )
      }
    }
  }

  private func toggleBookmark(for postID: String) {
    guard let index = posts.firstIndex(where: { $0.id == postID }) else { return }

    let post = posts[index]
    let shouldBookmark = !post.isBookmarked

    guard shouldBookmark else {
      presentRemoveBookmarkConfirmation(for: post)
      return
    }

    performBookmarkToggle(shouldBookmark: shouldBookmark, for: post)
  }

  private func performBookmarkToggle(shouldBookmark: Bool, for post: Post) {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    if !shouldBookmark, let index = posts.firstIndex(where: { $0.id == post.id }) {
      pendingRemovedBookmarkIndexes[post.id] = index
    }

    Task {
      do {
        _ = try await environment.bookmarkStore.setBookmarked(shouldBookmark, for: post)
        Haptics.shared.lightImpact()
      } catch {
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(
          shouldBookmark ? .saveBookmark : .removeBookmark
        )
      }
    }
  }

  private func presentRemoveBookmarkConfirmation(for post: Post) {
    let alert = UIAlertController(
      title: "Remove Bookmark?",
      message: "This post will be removed from your Bookmarks.",
      preferredStyle: .alert
    )
    alert.addAction(UIAlertAction(title: L10n.tr("common.cancel"), style: .cancel))
    alert.addAction(UIAlertAction(title: "Remove", style: .destructive) { [weak self] _ in
      self?.performBookmarkToggle(shouldBookmark: false, for: post)
    })
    present(alert, animated: true)
  }

  private func share(post: Post) {
    guard let url = environment.api.resolveURL(path: post.url) else { return }
    presentShareSheet(items: [url])
  }

  private func presentPostDetail(for post: Post) {
    navigationController?.pushViewController(
      PostDetailViewController(environment: environment, post: post),
      animated: true
    )
  }

  @objc private func handleBookmarkStateChange(_ notification: Notification) {
    guard let change = notification.bookmarkStateChange else { return }

    if change.post.isBookmarked {
      if let index = posts.firstIndex(where: { $0.id == change.post.id }) {
        posts[index] = change.post
        pendingRemovedBookmarkIndexes.removeValue(forKey: change.post.id)
        let indexPath = IndexPath(row: index, section: 0)
        if let cell = tableView.cellForRow(at: indexPath) as? PostTableViewCell {
          cell.configure(
            with: change.post,
            isBookmarkMutationInFlight: change.isInFlight
          )
        } else if isInVisibleHierarchy {
          tableView.reloadRows(at: [indexPath], with: .none)
        } else {
          needsTableReloadOnAppear = true
        }
      } else {
        let restoreIndex = min(pendingRemovedBookmarkIndexes.removeValue(forKey: change.post.id) ?? 0, posts.count)
        posts.insert(change.post, at: restoreIndex)
        updateBackgroundState()
        if isInVisibleHierarchy {
          tableView.performBatchUpdates({
            tableView.insertRows(at: [IndexPath(row: restoreIndex, section: 0)], with: .none)
          })
        } else {
          needsTableReloadOnAppear = true
        }
      }
    } else if let index = posts.firstIndex(where: { $0.id == change.post.id }) {
      posts.remove(at: index)
      updateBackgroundState()
      if isInVisibleHierarchy {
        tableView.performBatchUpdates({
          tableView.deleteRows(at: [IndexPath(row: index, section: 0)], with: .fade)
        })
      } else {
        needsTableReloadOnAppear = true
      }
    } else if !change.isInFlight {
      pendingRemovedBookmarkIndexes.removeValue(forKey: change.post.id)
    }
  }

  private func refreshRow(at index: Int) {
    let indexPath = IndexPath(row: index, section: 0)
    if let cell = tableView.cellForRow(at: indexPath) as? PostTableViewCell {
      cell.configure(
        with: posts[index],
        isBookmarkMutationInFlight: environment.bookmarkStore.isMutationInFlight(for: posts[index].id)
      )
    } else if isInVisibleHierarchy {
      tableView.reloadRows(at: [indexPath], with: .none)
    } else {
      needsTableReloadOnAppear = true
    }
  }

  private func shouldPresentOfflineLoadState(for error: Error) -> Bool {
    environment.connectivityMonitor.isOffline || (error as NSError).domain == NSURLErrorDomain
  }
}

extension ReadingListViewController: UITableViewDataSource, UITableViewDelegate {
  func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
    posts.count
  }

  func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
    guard let cell = tableView.dequeueReusableCell(
      withIdentifier: PostTableViewCell.reuseID,
      for: indexPath
    ) as? PostTableViewCell else {
      return UITableViewCell()
    }

    let post = posts[indexPath.row]
    cell.configure(
      with: post,
      isBookmarkMutationInFlight: environment.bookmarkStore.isMutationInFlight(for: post.id)
    )
    cell.onLikeTap = { [weak self] in
      self?.toggleLike(for: post.id)
    }
    cell.onShareTap = { [weak self] in
      self?.share(post: post)
    }
    cell.onBookmarkTap = { [weak self] in
      self?.toggleBookmark(for: post.id)
    }
    return cell
  }

  func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
    tableView.deselectRow(at: indexPath, animated: true)
    guard posts.indices.contains(indexPath.row) else { return }
    presentPostDetail(for: posts[indexPath.row])
  }

  func tableView(_ tableView: UITableView, willDisplay cell: UITableViewCell, forRowAt indexPath: IndexPath) {
    guard indexPath.row >= posts.count - 5 else { return }
    loadReadingList(reset: false)
  }
}
