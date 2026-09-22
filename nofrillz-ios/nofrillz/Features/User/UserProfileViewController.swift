import UIKit

private enum PostNotificationUserInfoKey {
  static let post = "post"
}

final class UserProfileViewController: BaseViewController {
  private enum LoadFailureState {
    case offline
    case generic

    var title: String {
      switch self {
      case .offline:
        "No internet connection"
      case .generic:
        "Couldn't load this profile."
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
  private let initialUser: User?
  private let showsLogoutButton: Bool
  private let useAuthenticatedUserEndpoint: Bool

  private var user: User?
  private var posts: [Post] = []
  private var postsNextCursor: String?
  private var isLoadingPosts = false
  private var isUpdatingRelationship = false
  private var inFlightLikePostIDs = Set<String>()
  private var loadFailureState: LoadFailureState?
  private var needsTableReloadOnAppear = false
  private var followersWidthConstraint: NSLayoutConstraint?
  private var followingWidthConstraint: NSLayoutConstraint?
  private var postsWidthConstraint: NSLayoutConstraint?
  private var countsBottomConstraint: NSLayoutConstraint?
  private var readingListTopConstraint: NSLayoutConstraint?
  private var readingListHeightConstraint: NSLayoutConstraint?
  private var readingListBottomConstraint: NSLayoutConstraint?

  private let navigationTitleContainer = UIView(frame: CGRect(x: 0, y: 0, width: 200, height: 34))

  private let navigationDefaultTitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.navigationTitle
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.textAlignment = .center
    label.lineBreakMode = .byTruncatingTail
    label.numberOfLines = 1
    label.text = L10n.tr("tab.profile")
    return label
  }()

  private let navigationTitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.editorialHeadline
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.textAlignment = .center
    label.lineBreakMode = .byTruncatingTail
    label.numberOfLines = 1
    return label
  }()

  private let navigationUsernameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.metadata
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .center
    label.lineBreakMode = .byTruncatingTail
    label.numberOfLines = 1
    return label
  }()

  private lazy var navigationTitleStack: UIStackView = {
    let stack = UIStackView(arrangedSubviews: [navigationTitleLabel, navigationUsernameLabel])
    stack.axis = .vertical
    stack.alignment = .center
    stack.spacing = 0
    stack.alpha = 0
    return stack
  }()

  private let avatarView = AvatarView()

  private let nameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.longFormHeadline
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 1
    label.lineBreakMode = .byTruncatingTail
    return label
  }()

  private let usernameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.username
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 1
    label.lineBreakMode = .byTruncatingTail
    return label
  }()

  private let relationLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.metadata
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    return label
  }()

  private let aboutLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.bio
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 0
    return label
  }()

  private lazy var followersButton: UIButton = {
    let button = makeCountPillButton()
    button.addTarget(self, action: #selector(followersTapped), for: .touchUpInside)
    return button
  }()

  private lazy var followingButton: UIButton = {
    let button = makeCountPillButton()
    button.addTarget(self, action: #selector(followingTapped), for: .touchUpInside)
    return button
  }()

  private lazy var postsCountButton: UIButton = {
    let button = makeCountPillButton()
    button.setTitle(L10n.tr("profile.posts_count", 0), for: .normal)
    button.addTarget(self, action: #selector(postsTapped), for: .touchUpInside)
    return button
  }()

  private lazy var readingListButton: UIButton = {
    let button = UIButton(type: .custom)
    button.translatesAutoresizingMaskIntoConstraints = false
    button.contentHorizontalAlignment = .fill
    button.backgroundColor = .clear
    button.addTarget(self, action: #selector(readingListTapped), for: .touchUpInside)
    button.accessibilityLabel = "Bookmarks"

    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.postBody
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = "Bookmarks"
    titleLabel.numberOfLines = 1
    titleLabel.textAlignment = .left

    let chevronView = UIImageView(image: AppIcon.chevronRight.image)
    chevronView.tintColor = Theme.secondaryText
    chevronView.contentMode = .scaleAspectFit
    chevronView.translatesAutoresizingMaskIntoConstraints = false
    NSLayoutConstraint.activate([
      chevronView.widthAnchor.constraint(equalToConstant: 28),
      chevronView.heightAnchor.constraint(equalToConstant: 28)
    ])

    let row = UIStackView(arrangedSubviews: [titleLabel, UIView(), chevronView])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = 12
    row.isUserInteractionEnabled = false
    row.translatesAutoresizingMaskIntoConstraints = false

    let separatorView = UIView()
    separatorView.translatesAutoresizingMaskIntoConstraints = false
    separatorView.backgroundColor = Theme.separator

    button.addSubview(row)
    button.addSubview(separatorView)
    NSLayoutConstraint.activate([
      row.topAnchor.constraint(equalTo: button.topAnchor, constant: 24),
      row.bottomAnchor.constraint(equalTo: button.bottomAnchor, constant: -24),
      row.leadingAnchor.constraint(equalTo: button.leadingAnchor),
      row.trailingAnchor.constraint(equalTo: button.trailingAnchor),

      separatorView.leadingAnchor.constraint(equalTo: button.leadingAnchor),
      separatorView.trailingAnchor.constraint(equalTo: button.trailingAnchor),
      separatorView.bottomAnchor.constraint(equalTo: button.bottomAnchor),
      separatorView.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale)
    ])

    return button
  }()

  private let tableView: UITableView = {
    let tableView = UITableView(frame: .zero, style: .plain)
    tableView.translatesAutoresizingMaskIntoConstraints = false
    tableView.separatorStyle = .none
    tableView.rowHeight = UITableView.automaticDimension
    tableView.estimatedRowHeight = 100
    return tableView
  }()

  private let inlineStatusView = InlineStatusView()

  private let headerContainer: UIView = {
    let view = UIView()
    view.translatesAutoresizingMaskIntoConstraints = false
    view.backgroundColor = Theme.background
    return view
  }()

  var resolvedUserID: String {
    if let user, !user.id.isEmpty {
      return user.id
    }
    return userID
  }

  init(
    environment: AppEnvironment,
    userID: String,
    initialUser: User? = nil,
    showsLogoutButton: Bool = false,
    useAuthenticatedUserEndpoint: Bool = false
  ) {
    self.environment = environment
    self.userID = userID
    self.initialUser = initialUser
    self.showsLogoutButton = showsLogoutButton
    self.useAuthenticatedUserEndpoint = useAuthenticatedUserEndpoint
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func viewDidLoad() {
    super.viewDidLoad()
    tabBarItem.title = nil
    configureNavigationTitleView()
    configureTrailingNavigationItem()
    setupTable()
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
    if showsLogoutButton {
      NotificationCenter.default.addObserver(
        self,
        selector: #selector(handlePostCreated(_:)),
        name: .postCreated,
        object: nil
      )
    }
    applyUser(initialUser)
    loadData()
  }

  override func viewDidAppear(_ animated: Bool) {
    super.viewDidAppear(animated)
    if needsTableReloadOnAppear {
      tableView.reloadData()
      needsTableReloadOnAppear = false
    }
  }

  override func viewDidLayoutSubviews() {
    super.viewDidLayoutSubviews()
    updateHeaderLayoutIfNeeded()
    updateNavigationTitleVisibility()
  }

  private func setupTable() {
    tableView.register(PostTableViewCell.self, forCellReuseIdentifier: PostTableViewCell.reuseID)
    tableView.dataSource = self
    tableView.delegate = self
    inlineStatusView.isHidden = true
    inlineStatusView.onActionTap = { [weak self] in
      self?.loadData()
    }
    tableView.backgroundView = inlineStatusView
    view.addSubview(tableView)

    NSLayoutConstraint.activate([
      tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor)
    ])

    NSLayoutConstraint.activate([
      avatarView.widthAnchor.constraint(equalToConstant: 72),
      avatarView.heightAnchor.constraint(equalToConstant: 72)
    ])

    let countsRow = UIView()
    countsRow.translatesAutoresizingMaskIntoConstraints = false
    followersButton.translatesAutoresizingMaskIntoConstraints = false
    followingButton.translatesAutoresizingMaskIntoConstraints = false
    postsCountButton.translatesAutoresizingMaskIntoConstraints = false
    countsRow.addSubview(followersButton)
    countsRow.addSubview(followingButton)
    countsRow.addSubview(postsCountButton)

    let nameStack = UIStackView(arrangedSubviews: [nameLabel, usernameLabel, relationLabel])
    nameStack.axis = .vertical
    nameStack.spacing = 2
    nameLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    usernameLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    relationLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

    let topRow = UIStackView(arrangedSubviews: [avatarView, nameStack])
    topRow.axis = .horizontal
    topRow.spacing = 12
    topRow.alignment = .center
    topRow.translatesAutoresizingMaskIntoConstraints = false
    nameStack.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    aboutLabel.translatesAutoresizingMaskIntoConstraints = false
    headerContainer.addSubview(topRow)
    headerContainer.addSubview(aboutLabel)
    headerContainer.addSubview(countsRow)
    headerContainer.addSubview(readingListButton)

    followersWidthConstraint = followersButton.widthAnchor.constraint(equalToConstant: 100)
    followingWidthConstraint = followingButton.widthAnchor.constraint(equalToConstant: 100)
    postsWidthConstraint = postsCountButton.widthAnchor.constraint(equalToConstant: 100)
    countsBottomConstraint = countsRow.bottomAnchor.constraint(equalTo: headerContainer.bottomAnchor, constant: -20)
    readingListTopConstraint = readingListButton.topAnchor.constraint(equalTo: countsRow.bottomAnchor, constant: 0)
    readingListHeightConstraint = readingListButton.heightAnchor.constraint(equalToConstant: 0)
    readingListBottomConstraint = readingListButton.bottomAnchor.constraint(equalTo: headerContainer.bottomAnchor)

    NSLayoutConstraint.activate([
      topRow.topAnchor.constraint(equalTo: headerContainer.topAnchor, constant: 20),
      topRow.leadingAnchor.constraint(equalTo: headerContainer.leadingAnchor, constant: 16),
      topRow.trailingAnchor.constraint(equalTo: headerContainer.trailingAnchor, constant: -16),

      aboutLabel.topAnchor.constraint(equalTo: topRow.bottomAnchor, constant: 15),
      aboutLabel.leadingAnchor.constraint(equalTo: headerContainer.leadingAnchor, constant: 16),
      aboutLabel.trailingAnchor.constraint(equalTo: headerContainer.trailingAnchor, constant: -16),

      countsRow.topAnchor.constraint(equalTo: aboutLabel.bottomAnchor, constant: 15),
      countsRow.leadingAnchor.constraint(equalTo: headerContainer.leadingAnchor, constant: 16),
      countsRow.trailingAnchor.constraint(equalTo: headerContainer.trailingAnchor, constant: -16),

      followersButton.leadingAnchor.constraint(equalTo: countsRow.leadingAnchor),
      followersButton.topAnchor.constraint(equalTo: countsRow.topAnchor),
      followersButton.bottomAnchor.constraint(equalTo: countsRow.bottomAnchor),

      followingButton.leadingAnchor.constraint(equalTo: followersButton.trailingAnchor, constant: 8),
      followingButton.topAnchor.constraint(equalTo: countsRow.topAnchor),
      followingButton.bottomAnchor.constraint(equalTo: countsRow.bottomAnchor),

      postsCountButton.leadingAnchor.constraint(equalTo: followingButton.trailingAnchor, constant: 8),
      postsCountButton.topAnchor.constraint(equalTo: countsRow.topAnchor),
      postsCountButton.bottomAnchor.constraint(equalTo: countsRow.bottomAnchor),
      postsCountButton.trailingAnchor.constraint(lessThanOrEqualTo: countsRow.trailingAnchor),
      readingListTopConstraint!,
      readingListButton.leadingAnchor.constraint(equalTo: headerContainer.leadingAnchor, constant: 16),
      readingListButton.trailingAnchor.constraint(equalTo: headerContainer.trailingAnchor, constant: -16),
      readingListHeightConstraint!,
      followersWidthConstraint!,
      followingWidthConstraint!,
      postsWidthConstraint!,
      countsBottomConstraint!
    ])

    headerContainer.frame = CGRect(x: 0, y: 0, width: tableView.bounds.width, height: 1)
    tableView.tableHeaderView = headerContainer
    refreshReadingListVisibility()
    updateHeaderLayoutIfNeeded()
    updateBackgroundState()
  }

  private func loadData() {
    loadFailureState = nil
    updateBackgroundState()
    Task {
      do {
        let resolvedUser: User
        if useAuthenticatedUserEndpoint {
          resolvedUser = try await environment.api.fetchCurrentUser()
          environment.sessionStore.updateUser(resolvedUser)
        } else {
          resolvedUser = try await environment.api.fetchUser(id: userID)
        }
        applyUser(resolvedUser)
        loadPosts(reset: true)
      } catch {
        if posts.isEmpty {
          loadFailureState = shouldPresentOfflineLoadState(for: error) ? .offline : .generic
          updateBackgroundState()
        } else {
          Haptics.shared.error()
          environment.feedbackCenter.showRecoverableActionFailure(.refreshProfile)
        }
      }
    }
  }

  private func loadPosts(reset: Bool) {
    guard !isLoadingPosts else { return }
    if !reset, postsNextCursor == nil { return }
    isLoadingPosts = true
    if reset {
      loadFailureState = nil
      updateBackgroundState()
    }

    Task {
      defer { isLoadingPosts = false }
      do {
        let page = try await environment.api.fetchPostsPage(
          for: userID,
          cursor: reset ? nil : postsNextCursor
        )
        let resolvedItems = environment.bookmarkStore.absorbServerPosts(page.items)
        if reset {
          posts = resolvedItems
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
        if let cursor = page.nextCursor {
          postsNextCursor = cursor
        } else if page.items.count >= 20, let lastPostID = page.items.last?.id {
          postsNextCursor = lastPostID
        } else {
          postsNextCursor = nil
        }
        refreshPostsCountLabel()
        loadFailureState = nil
        updateBackgroundState()
      } catch {
        if posts.isEmpty {
          loadFailureState = shouldPresentOfflineLoadState(for: error) ? .offline : .generic
          updateBackgroundState()
        } else if reset {
          Haptics.shared.error()
          environment.feedbackCenter.showRecoverableActionFailure(.refreshProfile)
        }
      }
    }
  }

  private func applyUser(_ user: User?) {
    guard let user else {
      nameLabel.text = L10n.tr("profile.default_username")
      let fallbackHandle = L10n.tr("profile.default_user_handle").trimmingCharacters(in: CharacterSet(charactersIn: "@"))
      usernameLabel.text = "@\(fallbackHandle)"
      navigationTitleLabel.text = L10n.tr("profile.default_username")
      navigationUsernameLabel.text = "@\(fallbackHandle)"
      navigationTitleStack.layoutIfNeeded()
      relationLabel.text = nil
      relationLabel.isHidden = true
      aboutLabel.text = nil
      avatarView.configure(
        username: L10n.tr("profile.default_user_handle"),
        firstName: nil,
        lastName: nil,
        avatarURL: nil
      )
      followersButton.setTitle(L10n.tr("profile.followers_count", 0), for: .normal)
      followingButton.setTitle(L10n.tr("profile.following_count", 0), for: .normal)
      refreshPostsCountLabel()
      refreshReadingListVisibility()
      return
    }

    self.user = user
    navigationTitleLabel.text = user.displayName
    navigationUsernameLabel.text = "@\(user.username)"
    navigationTitleStack.layoutIfNeeded()
    nameLabel.text = user.displayName
    usernameLabel.text = "@\(user.username)"
    avatarView.configure(
      username: user.username,
      firstName: user.firstName,
      lastName: user.lastName,
      avatarURL: user.avatarURL
    )
    relationLabel.text = showsLogoutButton ? nil : relationshipText(for: user)
    relationLabel.isHidden = relationLabel.text?.isEmpty ?? true
    aboutLabel.text = user.about ?? L10n.tr("profile.no_about")
    followersButton.setTitle(L10n.tr("profile.followers_count", user.followerCount), for: .normal)
    followingButton.setTitle(L10n.tr("profile.following_count", user.followingCount), for: .normal)
    refreshPostsCountLabel()
    refreshReadingListVisibility()
    updateHeaderLayoutIfNeeded()
    configureTrailingNavigationItem()
    updateNavigationTitleVisibility()
  }

  private func configureNavigationTitleView() {
    navigationDefaultTitleLabel.translatesAutoresizingMaskIntoConstraints = false
    navigationTitleStack.translatesAutoresizingMaskIntoConstraints = false
    navigationTitleContainer.addSubview(navigationDefaultTitleLabel)
    navigationTitleContainer.addSubview(navigationTitleStack)

    NSLayoutConstraint.activate([
      navigationDefaultTitleLabel.centerXAnchor.constraint(equalTo: navigationTitleContainer.centerXAnchor),
      navigationDefaultTitleLabel.centerYAnchor.constraint(equalTo: navigationTitleContainer.centerYAnchor),
      navigationDefaultTitleLabel.leadingAnchor.constraint(greaterThanOrEqualTo: navigationTitleContainer.leadingAnchor),
      navigationDefaultTitleLabel.trailingAnchor.constraint(lessThanOrEqualTo: navigationTitleContainer.trailingAnchor),

      navigationTitleStack.centerXAnchor.constraint(equalTo: navigationTitleContainer.centerXAnchor),
      navigationTitleStack.centerYAnchor.constraint(equalTo: navigationTitleContainer.centerYAnchor),
      navigationTitleStack.leadingAnchor.constraint(greaterThanOrEqualTo: navigationTitleContainer.leadingAnchor),
      navigationTitleStack.trailingAnchor.constraint(lessThanOrEqualTo: navigationTitleContainer.trailingAnchor)
    ])

    navigationItem.titleView = navigationTitleContainer
    navigationTitleStack.layoutIfNeeded()
    navigationDefaultTitleLabel.alpha = 1
  }

  private func makeCountPillButton() -> UIButton {
    let button = UIButton(type: .custom)
    button.titleLabel?.font = Theme.Typography.countPill
    button.titleLabel?.adjustsFontForContentSizeCategory = true
    button.titleLabel?.numberOfLines = 1
    button.titleLabel?.adjustsFontSizeToFitWidth = true
    button.titleLabel?.minimumScaleFactor = 0.75
    button.titleLabel?.lineBreakMode = .byTruncatingTail
    button.setTitleColor(Theme.text, for: .normal)
    button.contentEdgeInsets = UIEdgeInsets(top: 7, left: 10, bottom: 7, right: 10)
    button.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    Theme.applySurface(to: button, cornerRadius: 14, fillColor: Theme.elevatedSurfaceBackground)
    return button
  }

  private func refreshReadingListVisibility() {
    let shouldShow = isViewingOwnProfile
    readingListButton.isHidden = !shouldShow
    readingListButton.isUserInteractionEnabled = shouldShow
    readingListButton.alpha = shouldShow ? 1 : 0
    countsBottomConstraint?.isActive = !shouldShow
    readingListBottomConstraint?.isActive = shouldShow
    readingListTopConstraint?.constant = 0
    readingListHeightConstraint?.isActive = !shouldShow
  }

  private func configureTrailingNavigationItem() {
    if isViewingOwnProfile {
      navigationItem.rightBarButtonItem = makeNavigationIconBarButtonItem(
        image: AppIcon.settings.image,
        accessibilityLabel: L10n.tr("common.settings"),
        action: #selector(settingsTapped),
        iconSize: 28,
        buttonSize: 44
      )
      return
    }

    navigationItem.rightBarButtonItem = makeNavigationMenuBarButtonItem(
      image: AppIcon.more.image,
      accessibilityLabel: L10n.tr("common.more"),
      menu: makeMoreMenu(),
      iconSize: 28,
      buttonSize: 44
    )
  }

  private func makeMoreMenu() -> UIMenu {
    var actions: [UIAction] = []

    if !isViewingOwnProfile {
      let followTitle = user?.isFollowing == true ? L10n.tr("profile.unfollow_action") : L10n.tr("profile.follow_action")
      actions.append(UIAction(title: followTitle) { [weak self] _ in
        self?.toggleFollowFromMenu()
      })
    }

    actions.append(UIAction(title: L10n.tr("common.share")) { _ in })
    actions.append(UIAction(title: L10n.tr("common.report"), attributes: .destructive) { _ in })

    return UIMenu(children: actions)
  }

  private var isViewingOwnProfile: Bool {
    if showsLogoutButton {
      return true
    }
    guard let currentUserID = environment.sessionStore.currentUser?.id, !currentUserID.isEmpty else {
      return false
    }
    if let user, !user.id.isEmpty {
      return user.id == currentUserID
    }
    return userID == currentUserID
  }

  private func updateNavigationTitleVisibility() {
    guard isViewLoaded else { return }

    let nameFrame = nameLabel.convert(nameLabel.bounds, to: view)
    let thresholdY = tableView.frame.minY + 6
    let fadeDistance: CGFloat = 28
    let progress = max(0, min(1, (thresholdY - nameFrame.maxY) / fadeDistance))
    navigationDefaultTitleLabel.alpha = 1 - progress
    navigationTitleStack.alpha = progress
  }

  @objc private func settingsTapped() {
    let settingsVC = SettingsViewController(environment: environment)
    navigationController?.pushViewController(settingsVC, animated: true)
  }

  @objc private func readingListTapped() {
    navigationController?.pushViewController(
      ReadingListViewController(environment: environment),
      animated: true
    )
  }

  private func toggleFollowFromMenu() {
    guard !isViewingOwnProfile else { return }
    guard !isUpdatingRelationship else { return }
    guard let currentUserID = environment.sessionStore.currentUser?.id, !currentUserID.isEmpty else { return }
    guard let user, !user.id.isEmpty, user.id != currentUserID else { return }
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    let originalUser = user
    let shouldFollow = !user.isFollowing
    isUpdatingRelationship = true
    applyFollowState(shouldFollow, basedOn: originalUser)

    Task {
      defer { isUpdatingRelationship = false }

      do {
        if shouldFollow {
          try await environment.api.followUser(userID: user.id)
        } else {
          try await environment.api.unfollowUser(userID: user.id)
        }
        Haptics.shared.lightImpact()
      } catch {
        applyFollowState(originalUser.isFollowing, basedOn: originalUser)
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(shouldFollow ? .follow : .unfollow)
      }
    }
  }

  private func updateAuthenticatedUserFollowingCount(delta: Int) {
    environment.sessionStore.adjustCurrentUserFollowingCount(by: delta)
  }

  @objc private func handleFollowRelationshipChange(_ notification: Notification) {
    guard let change = notification.followRelationshipChange else { return }

    if let currentUser = user, currentUser.id == change.userID, !isViewingOwnProfile {
      guard currentUser.isFollowing != change.isFollowing else { return }
      let followerDelta = change.isFollowing ? 1 : -1
      let updatedFollowerCount = max(0, currentUser.followerCount + followerDelta)
      applyUser(currentUser.withRelationship(isFollowing: change.isFollowing, followerCount: updatedFollowerCount))
      return
    }

    guard isViewingOwnProfile, let currentUser = environment.sessionStore.currentUser else { return }
    guard let displayedUser = user, displayedUser.id == currentUser.id else { return }
    guard displayedUser.followingCount != currentUser.followingCount else { return }
    applyUser(
      User(
        id: displayedUser.id,
        username: displayedUser.username,
        firstName: displayedUser.firstName,
        lastName: displayedUser.lastName,
        email: displayedUser.email,
        about: displayedUser.about,
        avatarURL: displayedUser.avatarURL,
        joinedAt: displayedUser.joinedAt,
        isFollowing: displayedUser.isFollowing,
        isFollower: displayedUser.isFollower,
        followerCount: displayedUser.followerCount,
        followingCount: currentUser.followingCount,
        postCount: displayedUser.postCount
      )
    )
  }

  @objc private func handleBookmarkStateChange(_ notification: Notification) {
    guard let change = notification.bookmarkStateChange else { return }
    guard let index = posts.firstIndex(where: { $0.id == change.post.id }) else { return }

    posts[index] = change.post
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

  @objc private func handlePostCreated(_ notification: Notification) {
    guard showsLogoutButton else { return }

    guard let currentUserID = environment.sessionStore.currentUser?.id, !currentUserID.isEmpty else {
      loadPosts(reset: true)
      return
    }

    guard let post = notification.userInfo?[PostNotificationUserInfoKey.post] as? Post else {
      loadPosts(reset: true)
      return
    }

    guard post.authorID == currentUserID else { return }
    guard !posts.contains(where: { $0.id == post.id }) else { return }

    let resolvedPost = environment.bookmarkStore.absorbServerPosts([post]).first ?? post
    posts.insert(resolvedPost, at: 0)
    updateBackgroundState()

    if let existingUser = user {
      let updatedUser = User(
        id: existingUser.id,
        username: existingUser.username,
        firstName: existingUser.firstName,
        lastName: existingUser.lastName,
        email: existingUser.email,
        about: existingUser.about,
        avatarURL: existingUser.avatarURL,
        joinedAt: existingUser.joinedAt,
        isFollowing: existingUser.isFollowing,
        isFollower: existingUser.isFollower,
        followerCount: existingUser.followerCount,
        followingCount: existingUser.followingCount,
        postCount: existingUser.postCount + 1
      )
      user = updatedUser
      refreshPostsCountLabel()
    }

    if isInVisibleHierarchy {
      tableView.performBatchUpdates({
        tableView.insertRows(at: [IndexPath(row: 0, section: 0)], with: .none)
      })
    } else {
      needsTableReloadOnAppear = true
    }
  }

  private func relationshipText(for user: User) -> String {
    var tags: [String] = []
    if user.isFollowing {
      tags.append(L10n.tr("profile.you_follow"))
    }
    if user.isFollower {
      tags.append(L10n.tr("profile.follows_you"))
    }
    return tags.joined(separator: "  •  ")
  }

  private func refreshPostsCountLabel() {
    let displayedCount = max(user?.postCount ?? 0, posts.count)
    postsCountButton.setTitle(L10n.tr("profile.posts_count", displayedCount), for: .normal)
  }

  private func animatePostsBounce(duration: TimeInterval = 0.3) {
    postsCountButton.transform = .identity
    UIView.animate(withDuration: duration / 2, delay: 0, options: [.curveEaseOut]) {
      self.postsCountButton.transform = CGAffineTransform(scaleX: 1.035, y: 1.035)
    } completion: { _ in
      UIView.animate(withDuration: duration / 2, delay: 0, options: [.curveEaseIn]) {
        self.postsCountButton.transform = .identity
      }
    }
  }

  private func animateProfileWiggle(duration: TimeInterval = 0.32) {
    tableView.transform = .identity
    UIView.animateKeyframes(withDuration: duration, delay: 0, options: [.calculationModeCubic]) {
      UIView.addKeyframe(withRelativeStartTime: 0.0, relativeDuration: 0.33) {
        self.tableView.transform = CGAffineTransform(translationX: -8, y: 0)
      }
      UIView.addKeyframe(withRelativeStartTime: 0.33, relativeDuration: 0.33) {
        self.tableView.transform = CGAffineTransform(translationX: 8, y: 0)
      }
      UIView.addKeyframe(withRelativeStartTime: 0.66, relativeDuration: 0.34) {
        self.tableView.transform = .identity
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

  @objc private func followersTapped() {
    guard let user else { return }
    navigationController?.pushViewController(
      UserConnectionsViewController(environment: environment, userID: user.id, kind: .followers),
      animated: true
    )
  }

  @objc private func followingTapped() {
    guard let user else { return }
    navigationController?.pushViewController(
      UserConnectionsViewController(environment: environment, userID: user.id, kind: .following),
      animated: true
    )
  }

  @objc private func postsTapped() {
    animatePostsBounce()
    animateProfileWiggle()
  }

  private func applyFollowState(_ isFollowing: Bool, basedOn user: User) {
    let displayedUser = (self.user?.id == user.id ? self.user : nil) ?? user
    let followerDelta: Int
    if isFollowing == displayedUser.isFollowing {
      followerDelta = 0
    } else {
      followerDelta = isFollowing ? 1 : -1
    }
    let updatedFollowerCount = max(0, displayedUser.followerCount + followerDelta)
    let updatedUser = displayedUser.withRelationship(isFollowing: isFollowing, followerCount: updatedFollowerCount)
    applyUser(updatedUser)
    updateAuthenticatedUserFollowingCount(delta: followerDelta)
    NotificationCenter.default.post(
      name: .followRelationshipDidChange,
      object: FollowRelationshipChange(userID: displayedUser.id, isFollowing: isFollowing)
    )
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
    guard let loadFailureState, posts.isEmpty else {
      tableView.backgroundView?.isHidden = true
      return
    }

    inlineStatusView.configure(
      title: loadFailureState.title,
      message: loadFailureState.message,
      actionTitle: "Retry"
    )
    tableView.backgroundView?.isHidden = false
  }

  private func shouldPresentOfflineLoadState(for error: Error) -> Bool {
    environment.connectivityMonitor.isOffline || (error as NSError).domain == NSURLErrorDomain
  }
}

extension UserProfileViewController: UITableViewDataSource, UITableViewDelegate {
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
    loadPosts(reset: false)
  }

  func scrollViewDidScroll(_ scrollView: UIScrollView) {
    updateNavigationTitleVisibility()
  }

  private func updateHeaderLayoutIfNeeded() {
    guard tableView.tableHeaderView === headerContainer else { return }
    let width = max(tableView.bounds.width, 1)
    let horizontalPadding: CGFloat = 32
    let interItemSpacing: CGFloat = 16
    let computedButtonWidth = floor((width - horizontalPadding - interItemSpacing) / 3)
    followersWidthConstraint?.constant = max(72, computedButtonWidth)
    followingWidthConstraint?.constant = max(72, computedButtonWidth)
    postsWidthConstraint?.constant = max(72, computedButtonWidth)
    headerContainer.frame.size.width = width

    let fittingSize = CGSize(width: width, height: UIView.layoutFittingCompressedSize.height)
    let size = headerContainer.systemLayoutSizeFitting(
      fittingSize,
      withHorizontalFittingPriority: .required,
      verticalFittingPriority: .fittingSizeLevel
    )
    let height = ceil(size.height)

    guard headerContainer.frame.height != height || headerContainer.frame.width != width else {
      return
    }

    headerContainer.frame = CGRect(x: 0, y: 0, width: width, height: height)
    tableView.tableHeaderView = headerContainer
  }
}
