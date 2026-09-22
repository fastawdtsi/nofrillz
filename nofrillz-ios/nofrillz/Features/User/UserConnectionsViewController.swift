import UIKit

@MainActor
final class UserConnectionsViewController: BaseViewController {
  enum Kind {
    case followers
    case following

    var title: String {
      switch self {
      case .followers: return L10n.tr("profile.followers")
      case .following: return L10n.tr("profile.following")
      }
    }
  }

  private enum LoadFailureState {
    case offline
    case generic

    var title: String {
      switch self {
      case .offline:
        "No internet connection"
      case .generic:
        "Couldn't load this list."
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
  private let userID: String
  private let kind: Kind

  private var users: [User] = []
  private var followingIDs = Set<String>()
  private var nextCursor: String?
  private var isLoadingPage = false
  private var inFlightFollowUserIDs = Set<String>()
  private var loadFailureState: LoadFailureState?
  private var needsTableReloadOnAppear = false

  private let tableView: UITableView = {
    let tableView = UITableView(frame: .zero, style: .plain)
    tableView.translatesAutoresizingMaskIntoConstraints = false
    tableView.separatorStyle = .none
    return tableView
  }()

  private let emptyLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .center
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()

  private let inlineStatusView = InlineStatusView()

  private let navigationTitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.navigationTitle
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.textAlignment = .center
    return label
  }()

  init(environment: AppEnvironment, userID: String, kind: Kind) {
    self.environment = environment
    self.userID = userID
    self.kind = kind
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func viewDidLoad() {
    super.viewDidLoad()
    title = nil
    navigationTitleLabel.text = kind.title
    navigationTitleLabel.sizeToFit()
    navigationItem.titleView = navigationTitleLabel

    tableView.register(UserCell.self, forCellReuseIdentifier: UserCell.reuseID)
    tableView.dataSource = self
    tableView.delegate = self
    inlineStatusView.isHidden = true
    inlineStatusView.onActionTap = { [weak self] in
      self?.loadUsers(reset: true)
    }
    view.addSubview(tableView)
    view.addSubview(emptyLabel)
    view.addSubview(inlineStatusView)

    NSLayoutConstraint.activate([
      tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),

      emptyLabel.centerXAnchor.constraint(equalTo: view.centerXAnchor),
      emptyLabel.centerYAnchor.constraint(equalTo: view.centerYAnchor),

      inlineStatusView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      inlineStatusView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      inlineStatusView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      inlineStatusView.trailingAnchor.constraint(equalTo: view.trailingAnchor)
    ])

    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleFollowRelationshipChange(_:)),
      name: .followRelationshipDidChange,
      object: nil
    )

    loadUsers(reset: true)
  }

  override func viewDidAppear(_ animated: Bool) {
    super.viewDidAppear(animated)
    if needsTableReloadOnAppear {
      render()
      needsTableReloadOnAppear = false
    }
  }

  private func loadUsers(reset: Bool) {
    guard !isLoadingPage else { return }
    if !reset, nextCursor == nil { return }
    isLoadingPage = true
    if reset {
      loadFailureState = nil
      render()
    }

    Task {
      defer { isLoadingPage = false }
      do {
        let page: CursorPage<User>
        switch kind {
        case .followers:
          page = try await environment.api.fetchFollowersPage(for: userID, cursor: reset ? nil : nextCursor)
        case .following:
          page = try await environment.api.fetchFollowingPage(for: userID, cursor: reset ? nil : nextCursor)
        }
        if reset {
          users = page.items
          followingIDs = Set(page.items.filter(\.isFollowing).map(\.id))
          if isInVisibleHierarchy {
            render()
          } else {
            needsTableReloadOnAppear = true
          }
        } else {
          let start = users.count
          users.append(contentsOf: page.items)
          page.items.filter(\.isFollowing).forEach { followingIDs.insert($0.id) }
          emptyLabel.isHidden = !users.isEmpty
          emptyLabel.text = users.isEmpty ? L10n.tr("profile.no_users_yet") : nil
          if !page.items.isEmpty {
            if isInVisibleHierarchy {
              let indexPaths = (start ..< users.count).map { IndexPath(row: $0, section: 0) }
              tableView.performBatchUpdates({
                tableView.insertRows(at: indexPaths, with: .none)
              })
            } else {
              needsTableReloadOnAppear = true
            }
          }
        }
        nextCursor = page.nextCursor
        loadFailureState = nil
        render()
      } catch {
        if users.isEmpty {
          loadFailureState = shouldPresentOfflineState(for: error) ? .offline : .generic
          render()
        }
      }
    }
  }

  private func render() {
    if let loadFailureState, users.isEmpty {
      inlineStatusView.isHidden = false
      inlineStatusView.configure(
        title: loadFailureState.title,
        message: loadFailureState.message,
        actionTitle: "Retry"
      )
      emptyLabel.isHidden = true
    } else {
      inlineStatusView.isHidden = true
      emptyLabel.isHidden = !users.isEmpty
      emptyLabel.text = users.isEmpty ? L10n.tr("profile.no_users_yet") : nil
    }
    tableView.reloadData()
  }

  private func toggleFollow(userID: String) {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }
    guard !inFlightFollowUserIDs.contains(userID) else { return }
    guard let row = users.firstIndex(where: { $0.id == userID }) else { return }
    let user = users[row]
    guard user.id != environment.sessionStore.currentUser?.id else { return }
    let isFollowing = followingIDs.contains(user.id) || user.isFollowing
    let shouldFollow = !isFollowing
    inFlightFollowUserIDs.insert(userID)
    applyFollowState(shouldFollow, to: row)
    Task {
      do {
        if isFollowing {
          try await environment.api.unfollowUser(userID: user.id)
        } else {
          try await environment.api.followUser(userID: user.id)
        }
        inFlightFollowUserIDs.remove(userID)
        Haptics.shared.lightImpact()
      } catch {
        inFlightFollowUserIDs.remove(userID)
        guard let updatedRow = users.firstIndex(where: { $0.id == userID }) else { return }
        applyFollowState(isFollowing, to: updatedRow)
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(shouldFollow ? .follow : .unfollow)
      }
    }
  }

  @objc private func handleFollowRelationshipChange(_ notification: Notification) {
    guard let change = notification.followRelationshipChange else { return }
    guard let row = users.firstIndex(where: { $0.id == change.userID }) else { return }

    let updatedUser = users[row].withFollowing(change.isFollowing)
    guard updatedUser != users[row] || followingIDs.contains(change.userID) != change.isFollowing else { return }

    users[row] = updatedUser
    if change.isFollowing {
      followingIDs.insert(change.userID)
    } else {
      followingIDs.remove(change.userID)
    }

    let indexPath = IndexPath(row: row, section: 0)
    if isInVisibleHierarchy {
      if let cell = tableView.cellForRow(at: indexPath) as? UserCell {
        let followEnabled = updatedUser.id != environment.sessionStore.currentUser?.id
        cell.configure(user: updatedUser, isFollowing: change.isFollowing, followEnabled: followEnabled)
      } else {
        tableView.reloadRows(at: [indexPath], with: .none)
      }
    } else {
      needsTableReloadOnAppear = true
    }
  }

  private func applyFollowState(_ isFollowing: Bool, to row: Int) {
    let existingUser = users[row]
    let wasFollowing = followingIDs.contains(existingUser.id) || existingUser.isFollowing
    users[row] = existingUser.withFollowing(isFollowing)
    let delta: Int
    if isFollowing == wasFollowing {
      delta = 0
    } else {
      delta = isFollowing ? 1 : -1
    }
    if isFollowing {
      followingIDs.insert(existingUser.id)
    } else {
      followingIDs.remove(existingUser.id)
    }
    environment.sessionStore.adjustCurrentUserFollowingCount(by: delta)

    NotificationCenter.default.post(
      name: .followRelationshipDidChange,
      object: FollowRelationshipChange(userID: existingUser.id, isFollowing: isFollowing)
    )

    let indexPath = IndexPath(row: row, section: 0)
    if isInVisibleHierarchy {
      if let cell = tableView.cellForRow(at: indexPath) as? UserCell {
        let updatedUser = users[row]
        let followEnabled = updatedUser.id != environment.sessionStore.currentUser?.id
        cell.configure(user: updatedUser, isFollowing: isFollowing, followEnabled: followEnabled)
      } else {
        tableView.reloadRows(at: [indexPath], with: .none)
      }
    } else {
      needsTableReloadOnAppear = true
    }
  }

  private func shouldPresentOfflineState(for error: Error) -> Bool {
    environment.connectivityMonitor.isOffline || (error as NSError).domain == NSURLErrorDomain
  }
}

extension UserConnectionsViewController: UITableViewDataSource {
  func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
    users.count
  }

  func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
    guard let cell = tableView.dequeueReusableCell(
      withIdentifier: UserCell.reuseID,
      for: indexPath
    ) as? UserCell else {
      return UITableViewCell()
    }

    let user = users[indexPath.row]
    let isFollowing = followingIDs.contains(user.id) || user.isFollowing
    let followEnabled = user.id != environment.sessionStore.currentUser?.id
    cell.configure(user: user, isFollowing: isFollowing, followEnabled: followEnabled)
    cell.onFollowTap = { [weak self] in
      self?.toggleFollow(userID: user.id)
    }
    return cell
  }
}

extension UserConnectionsViewController: UITableViewDelegate {
  func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
    tableView.deselectRow(at: indexPath, animated: true)
    let user = users[indexPath.row]
    navigationController?.pushViewController(
      UserProfileViewController(environment: environment, userID: user.id, initialUser: user),
      animated: true
    )
  }

  func tableView(_ tableView: UITableView, willDisplay cell: UITableViewCell, forRowAt indexPath: IndexPath) {
    guard indexPath.row >= users.count - 5 else { return }
    loadUsers(reset: false)
  }
}
