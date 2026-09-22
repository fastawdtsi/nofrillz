import UIKit

final class FeedViewController: BaseViewController {
  private enum Constants {
    static let statusBarChromeFadeHeight: CGFloat = 28
    static let cachedPostLimit = 20
  }

  private enum LoadFailureState {
    case offline
    case generic

    var title: String {
      switch self {
      case .offline:
        "No internet connection"
      case .generic:
        "Couldn't load the feed."
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

  private let discover: Bool
  private let environment: AppEnvironment
  private let feedCacheStore = FeedCacheStore()
  private var posts: [Post] = []
  private var nextCursor: String?
  private var isLoadingPage = false
  private var inFlightLikePostIDs = Set<String>()
  private var needsTableReloadOnAppear = false
  private var lastScrollOffsetY: CGFloat = 0
  private var floatingControlsHidden = false
  private var didTriggerRefreshThresholdHaptic = false
  private var needsFeedRefreshOnAppear = false
  private var loadFailureState: LoadFailureState?
  private let statusBarChromeView = UIView()
  private let statusBarChromeGradientLayer = CAGradientLayer()
  private var statusBarChromeHeightConstraint: NSLayoutConstraint?
  private let bannerHeaderContainer = UIView()
  private let bannerImageView = UIImageView()
  private var bannerTopConstraint: NSLayoutConstraint?
  private var bannerHeightConstraint: NSLayoutConstraint?
  private let inlineStatusView = InlineStatusView()

  var onFloatingControlsVisibilityChange: ((Bool, Bool) -> Void)?

  private let tableView: UITableView = {
    let tableView = UITableView(frame: .zero, style: .plain)
    tableView.translatesAutoresizingMaskIntoConstraints = false
    tableView.separatorStyle = .none
    tableView.rowHeight = UITableView.automaticDimension
    tableView.estimatedRowHeight = 120
    return tableView
  }()

  private let refreshControl = UIRefreshControl()

  init(environment: AppEnvironment, discover: Bool = false) {
    self.discover = discover
    self.environment = environment
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func viewDidLoad() {
    super.viewDidLoad()
    if discover {
      title = "Discover"
      navigationItem.rightBarButtonItem = UIBarButtonItem(title: "Find people", style: .plain, target: self, action: #selector(findPeopleTapped))
    }

    setupTable()
    loadCachedFeedIfAvailable()

    NotificationCenter.default.addObserver(
      self,
      selector: #selector(refreshFromNotification),
      name: .postCreated,
      object: nil
    )
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleFollowRelationshipChange(_:)),
      name: .followRelationshipDidChange,
      object: nil
    )
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleBookmarkStateChange(_:)),
      name: .bookmarkStateDidChange,
      object: nil
    )

    loadFeed(reset: true)
  }

  override func viewDidAppear(_ animated: Bool) {
    super.viewDidAppear(animated)
    lastScrollOffsetY = tableView.contentOffset.y
    if needsFeedRefreshOnAppear {
      needsFeedRefreshOnAppear = false
      loadFeed(reset: true)
    }
    if needsTableReloadOnAppear {
      tableView.reloadData()
      needsTableReloadOnAppear = false
    }
  }

  override func viewWillAppear(_ animated: Bool) {
    super.viewWillAppear(animated)
    navigationController?.setNavigationBarHidden(!discover, animated: animated)
  }

  override func viewWillDisappear(_ animated: Bool) {
    super.viewWillDisappear(animated)
    navigationController?.setNavigationBarHidden(false, animated: animated)
  }

  override func viewDidLayoutSubviews() {
    super.viewDidLayoutSubviews()
    updateBannerHeaderFrameIfNeeded()
    updateBottomInsets()
    updateStatusBarChromeGradient()
  }

  @objc private func findPeopleTapped() {
    navigationController?.pushViewController(SearchViewController(environment: environment), animated: true)
  }

  private func setupTable() {
    tableView.dataSource = self
    tableView.delegate = self
    tableView.register(PostTableViewCell.self, forCellReuseIdentifier: PostTableViewCell.reuseID)
    tableView.contentInsetAdjustmentBehavior = .never
    tableView.refreshControl = refreshControl
    inlineStatusView.isHidden = true
    inlineStatusView.onActionTap = { [weak self] in
      self?.loadFeed(reset: true)
    }
    tableView.backgroundView = inlineStatusView
    refreshControl.addTarget(self, action: #selector(refreshTapped), for: .valueChanged)
    if !discover { configureBannerHeader() }
    configureStatusBarChrome()

    view.addSubview(tableView)
    view.addSubview(statusBarChromeView)

    NSLayoutConstraint.activate([
      tableView.topAnchor.constraint(equalTo: discover ? view.safeAreaLayoutGuide.topAnchor : view.topAnchor),
      tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
      statusBarChromeView.topAnchor.constraint(equalTo: view.topAnchor),
      statusBarChromeView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      statusBarChromeView.trailingAnchor.constraint(equalTo: view.trailingAnchor)
    ])

    statusBarChromeHeightConstraint = statusBarChromeView.heightAnchor.constraint(equalToConstant: 0)
    statusBarChromeHeightConstraint?.isActive = true
  }

  private func configureBannerHeader() {
    bannerHeaderContainer.backgroundColor = .clear

    bannerImageView.translatesAutoresizingMaskIntoConstraints = false
    bannerImageView.image = UIImage(named: "FeedBanner")?.withRenderingMode(.alwaysTemplate)
    bannerImageView.tintColor = Theme.text
    bannerImageView.contentMode = .scaleAspectFit
    bannerImageView.clipsToBounds = true

    bannerHeaderContainer.addSubview(bannerImageView)

    bannerTopConstraint = bannerImageView.topAnchor.constraint(equalTo: bannerHeaderContainer.topAnchor, constant: 0)
    bannerHeightConstraint = bannerImageView.heightAnchor.constraint(equalToConstant: 100)

    NSLayoutConstraint.activate([
      bannerTopConstraint!,
      bannerHeightConstraint!,
      bannerImageView.leadingAnchor.constraint(equalTo: bannerHeaderContainer.leadingAnchor),
      bannerImageView.trailingAnchor.constraint(equalTo: bannerHeaderContainer.trailingAnchor)
    ])

    tableView.tableHeaderView = bannerHeaderContainer
    updateBannerHeaderFrameIfNeeded()
  }

  private func configureStatusBarChrome() {
    statusBarChromeView.translatesAutoresizingMaskIntoConstraints = false
    statusBarChromeView.isUserInteractionEnabled = false
    statusBarChromeView.alpha = 1
    statusBarChromeView.backgroundColor = .clear
    statusBarChromeGradientLayer.startPoint = CGPoint(x: 0.5, y: 0)
    statusBarChromeGradientLayer.endPoint = CGPoint(x: 0.5, y: 1)
    statusBarChromeView.layer.addSublayer(statusBarChromeGradientLayer)
  }

  private func updateStatusBarChromeGradient() {
    statusBarChromeGradientLayer.frame = statusBarChromeView.bounds
    let backgroundColor = Theme.background.resolvedColor(with: traitCollection)
    statusBarChromeGradientLayer.colors = [
      backgroundColor.cgColor,
      backgroundColor.withAlphaComponent(0).cgColor
    ]
    statusBarChromeGradientLayer.locations = [0, 1]
  }

  private func updateBannerHeaderFrameIfNeeded() {
    guard tableView.tableHeaderView === bannerHeaderContainer else { return }

    let width = tableView.bounds.width
    guard width > 0 else { return }

    let rawImageHeight: CGFloat
    if let image = bannerImageView.image, image.size.width > 0 {
      rawImageHeight = width * (image.size.height / image.size.width)
    } else {
      rawImageHeight = 140
    }

    let topPadding = view.safeAreaInsets.top + 8
    let scaledImageHeight = min(max(rawImageHeight * 0.5, 40), 110)
    bannerTopConstraint?.constant = topPadding
    bannerHeightConstraint?.constant = scaledImageHeight

    let height = topPadding + scaledImageHeight + 8
    if bannerHeaderContainer.frame.size.width != width || bannerHeaderContainer.frame.size.height != height {
      bannerHeaderContainer.frame = CGRect(x: 0, y: 0, width: width, height: height)
      tableView.tableHeaderView = bannerHeaderContainer
    }

    statusBarChromeHeightConstraint?.constant = view.safeAreaInsets.top + Constants.statusBarChromeFadeHeight
  }

  private func updateBottomInsets() {
    let bottomInset: CGFloat

    if let mainTabBarController = parentMainController {
      bottomInset = mainTabBarController.rootContentBottomInset
    } else {
      bottomInset = view.safeAreaInsets.bottom + 20
    }

    if tableView.contentInset.bottom != bottomInset {
      tableView.contentInset.bottom = bottomInset
      tableView.verticalScrollIndicatorInsets.bottom = bottomInset
    }
  }

  private var parentMainController: MainTabBarController? {
    var currentParent = parent
    while let current = currentParent {
      if let controller = current as? MainTabBarController {
        return controller
      }
      currentParent = current.parent
    }
    return nil
  }

  private func setFloatingControlsHidden(_ hidden: Bool, animated: Bool) {
    guard floatingControlsHidden != hidden else { return }
    floatingControlsHidden = hidden
    onFloatingControlsVisibilityChange?(hidden, animated)
  }

  private func maybeTriggerRefreshThresholdHaptic(for scrollView: UIScrollView) {
    guard scrollView.isDragging else { return }
    guard !refreshControl.isRefreshing else { return }
    guard !didTriggerRefreshThresholdHaptic else { return }

    let pullDistance = max(0, -(scrollView.contentOffset.y + scrollView.adjustedContentInset.top))
    let refreshThreshold = max(refreshControl.bounds.height, 80)
    guard pullDistance >= refreshThreshold else { return }

    didTriggerRefreshThresholdHaptic = true
    Haptics.shared.selection()
  }

  @objc private func refreshFromNotification() {
    loadFeed(reset: true)
  }

  @objc private func handleFollowRelationshipChange(_ notification: Notification) {
    guard notification.followRelationshipChange != nil else { return }
    if isLoadingPage {
      needsFeedRefreshOnAppear = true
      return
    }
    if isInVisibleHierarchy {
      loadFeed(reset: true)
    } else {
      needsFeedRefreshOnAppear = true
    }
  }

  @objc private func refreshTapped() {
    loadFeed(reset: true)
  }

  @objc private func handleBookmarkStateChange(_ notification: Notification) {
    guard let change = notification.bookmarkStateChange else { return }
    guard let index = posts.firstIndex(where: { $0.id == change.post.id }) else { return }

    posts[index] = change.post
    persistFeedCache()
    updateBackgroundState()

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
  }

  private func loadFeed(reset: Bool) {
    guard !isLoadingPage else { return }
    if !reset, nextCursor == nil { return }
    isLoadingPage = true
    let hadVisiblePosts = reset && !posts.isEmpty
    if reset {
      loadFailureState = nil
      updateBackgroundState()
    }

    Task {
      defer {
        isLoadingPage = false
        refreshControl.endRefreshing()
        updateBackgroundState()
      }
      do {
        let page = try await environment.api.fetchFeedPage(cursor: reset ? nil : nextCursor, discover: discover)
        let resolvedItems = environment.bookmarkStore.absorbServerPosts(page.items)
        if reset {
          posts = resolvedItems
          loadFailureState = nil
          persistFeedCache()
          if isInVisibleHierarchy {
            tableView.reloadData()
          } else {
            needsTableReloadOnAppear = true
          }
        } else {
          let start = posts.count
          posts.append(contentsOf: resolvedItems)
          if !resolvedItems.isEmpty {
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
          updateBackgroundState()
        } else if reset || refreshControl.isRefreshing {
          Haptics.shared.error()
          environment.feedbackCenter.showRecoverableActionFailure(.refreshFeed)
        }
      }
    }
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
    persistFeedCache()

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
        persistFeedCache()
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(
          desiredLikedState ? .respect : .removeRespect
        )
      }
    }
  }

  private func share(post: Post) {
    guard let url = environment.api.resolveURL(path: post.url) else { return }
    presentShareSheet(items: [url])
  }

  private func toggleBookmark(for postID: String) {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }
    guard let index = posts.firstIndex(where: { $0.id == postID }) else { return }

    let post = posts[index]
    let shouldBookmark = !post.isBookmarked

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

  private func presentPostDetail(for post: Post) {
    navigationController?.pushViewController(
      PostDetailViewController(environment: environment, post: post),
      animated: true
    )
  }

  private func loadCachedFeedIfAvailable() {
    guard !discover else { return }
    guard let userID = environment.sessionStore.currentUser?.id,
          let cachedPosts = feedCacheStore.loadPosts(for: userID),
          !cachedPosts.isEmpty else {
      updateBackgroundState()
      return
    }

    posts = environment.bookmarkStore.absorbServerPosts(cachedPosts)
    tableView.reloadData()
    updateBackgroundState()
  }

  private func persistFeedCache() {
    guard !discover else { return }
    guard let userID = environment.sessionStore.currentUser?.id else { return }
    feedCacheStore.save(posts: Array(posts.prefix(Constants.cachedPostLimit)), for: userID)
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

  private func updateBackgroundState() {
    guard let failureState = loadFailureState, posts.isEmpty else {
      tableView.backgroundView?.isHidden = true
      return
    }

    inlineStatusView.configure(
      title: failureState.title,
      message: failureState.message,
      actionTitle: "Retry"
    )
    tableView.backgroundView?.isHidden = false
  }

  private func shouldPresentOfflineLoadState(for error: Error) -> Bool {
    environment.connectivityMonitor.isOffline || (error as NSError).domain == NSURLErrorDomain
  }
}

extension FeedViewController: UITableViewDataSource, UITableViewDelegate {
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

  func scrollViewDidScroll(_ scrollView: UIScrollView) {
    maybeTriggerRefreshThresholdHaptic(for: scrollView)

    let offsetY = scrollView.contentOffset.y
    defer { lastScrollOffsetY = offsetY }

    let topBoundary = -scrollView.adjustedContentInset.top
    if offsetY <= topBoundary + 12 {
      setFloatingControlsHidden(false, animated: true)
      return
    }

    let canScroll = scrollView.contentSize.height + scrollView.adjustedContentInset.top + scrollView.adjustedContentInset.bottom
      > scrollView.bounds.height + 1
    guard canScroll else {
      setFloatingControlsHidden(false, animated: true)
      return
    }

    guard scrollView.isDragging || scrollView.isDecelerating else { return }

    let delta = offsetY - lastScrollOffsetY
    guard abs(delta) > 2 else { return }

    if delta > 0 {
      setFloatingControlsHidden(true, animated: true)
    } else {
      setFloatingControlsHidden(false, animated: true)
    }
  }

  func scrollViewDidEndDragging(_ scrollView: UIScrollView, willDecelerate decelerate: Bool) {
    didTriggerRefreshThresholdHaptic = false
  }

  func tableView(_ tableView: UITableView, willDisplay cell: UITableViewCell, forRowAt indexPath: IndexPath) {
    guard indexPath.row >= posts.count - 5 else { return }
    loadFeed(reset: false)
  }
}

private final class FeedCacheStore {
  private enum Keys {
    static let prefix = "nofrillz.feedCache."
  }

  private let defaults: UserDefaults

  init(defaults: UserDefaults = .standard) {
    self.defaults = defaults
  }

  func loadPosts(for userID: String) -> [Post]? {
    guard let data = defaults.data(forKey: cacheKey(for: userID)) else { return nil }

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601

    guard let payload = try? decoder.decode(CachedFeedPayload.self, from: data) else {
      defaults.removeObject(forKey: cacheKey(for: userID))
      return nil
    }

    return payload.posts.map(\.post)
  }

  func save(posts: [Post], for userID: String) {
    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601

    let payload = CachedFeedPayload(posts: posts.map(CachedPost.init))
    guard let data = try? encoder.encode(payload) else { return }
    defaults.set(data, forKey: cacheKey(for: userID))
  }

  private func cacheKey(for userID: String) -> String {
    "\(Keys.prefix)\(userID)"
  }
}

private struct CachedFeedPayload: Codable {
  let posts: [CachedPost]
}

private struct CachedPost: Codable {
  let id: String
  let url: String?
  let authorID: String
  let authorUsername: String
  let authorFirstName: String?
  let authorLastName: String?
  let authorAvatarURL: String?
  let content: String
  let createdAt: Date?
  let source: String
  let liked: Bool
  let isBookmarked: Bool
  let bookmarkedAt: Date?
  let likeCount: Int
  let commentCount: Int

  enum CodingKeys: String, CodingKey {
    case id
    case url
    case authorID
    case authorUsername
    case authorFirstName
    case authorLastName
    case authorAvatarURL
    case content
    case createdAt
    case source
    case liked
    case isBookmarked
    case bookmarkedAt
    case likeCount
    case commentCount
  }

  init(post: Post) {
    id = post.id
    url = post.url
    authorID = post.authorID
    authorUsername = post.authorUsername
    authorFirstName = post.authorFirstName
    authorLastName = post.authorLastName
    authorAvatarURL = post.authorAvatarURL
    content = post.content
    createdAt = post.createdAt
    source = post.source
    liked = post.liked
    isBookmarked = post.isBookmarked
    bookmarkedAt = post.bookmarkedAt
    likeCount = post.likeCount
    commentCount = post.commentCount
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    id = try container.decode(String.self, forKey: .id)
    url = try container.decodeIfPresent(String.self, forKey: .url)
    authorID = try container.decode(String.self, forKey: .authorID)
    authorUsername = try container.decode(String.self, forKey: .authorUsername)
    authorFirstName = try container.decodeIfPresent(String.self, forKey: .authorFirstName)
    authorLastName = try container.decodeIfPresent(String.self, forKey: .authorLastName)
    authorAvatarURL = try container.decodeIfPresent(String.self, forKey: .authorAvatarURL)
    content = try container.decode(String.self, forKey: .content)
    createdAt = try container.decodeIfPresent(Date.self, forKey: .createdAt)
    source = try container.decode(String.self, forKey: .source)
    liked = try container.decode(Bool.self, forKey: .liked)
    isBookmarked = try container.decodeIfPresent(Bool.self, forKey: .isBookmarked) ?? false
    bookmarkedAt = try container.decodeIfPresent(Date.self, forKey: .bookmarkedAt)
    likeCount = try container.decode(Int.self, forKey: .likeCount)
    commentCount = try container.decode(Int.self, forKey: .commentCount)
  }

  var post: Post {
    Post(
      id: id,
      url: url,
      authorID: authorID,
      authorUsername: authorUsername,
      authorFirstName: authorFirstName,
      authorLastName: authorLastName,
      authorAvatarURL: authorAvatarURL,
      content: content,
      createdAt: createdAt,
      source: source,
      liked: liked,
      isBookmarked: isBookmarked,
      bookmarkedAt: bookmarkedAt,
      likeCount: likeCount,
      commentCount: commentCount
    )
  }
}
