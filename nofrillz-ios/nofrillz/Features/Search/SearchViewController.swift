import UIKit

@MainActor
final class SearchViewController: BaseViewController {
  private enum Constants {
    static let horizontalInset: CGFloat = 16
    static let sectionSpacing: CGFloat = 32
    static let rowSpacing: CGFloat = 0
    static let searchFieldHeight: CGFloat = 56
    static let recentSearchLimit = 5
    static let recentSearchesKey = "nofrillz.search.recentQueries"
  }

  private enum SearchFailureState {
    case offline
    case generic

    var message: String {
      switch self {
      case .offline:
        "No internet connection.\nWe'll keep trying to reconnect."
      case .generic:
        "Couldn't load search results.\nTry again in a moment."
      }
    }
  }

  private enum Copy {
    static let title = localized("discover.title", fallback: "Discover")
    static let searchPlaceholder = localized(
      "discover.search_placeholder",
      fallback: "Search people, topics, or writing"
    )
    static let topicsTitle = localized("discover.topics.title", fallback: "Topics")
    static let topicsSubtitle = localized(
      "discover.topics.subtitle",
      fallback: "Explore ideas and conversations."
    )
    static let seeAll = localized("discover.topics.see_all", fallback: "See all")
    static let showLess = localized("discover.topics.show_less", fallback: "Show less")
    static let recentSearches = localized("discover.recent_searches.title", fallback: "Recent Searches")
    static let clearAll = localized("discover.recent_searches.clear_all", fallback: "Clear all")
    static let recentEmpty = localized(
      "discover.recent_searches.empty",
      fallback: "Search history will show here."
    )

    private static func localized(_ key: String, fallback: String) -> String {
      let value = L10n.tr(key)
      return value == key ? fallback : value
    }
  }

  private static let fallbackTopics = [
    Topic(
      name: "Technology",
      description: "Product launches, startups, gadgets, and the ideas shaping what comes next."
    ),
    Topic(
      name: "Design",
      description: "Brand systems, interiors, visual culture, and the details that make things feel intentional."
    ),
    Topic(
      name: "Travel",
      description: "Destinations, neighborhood finds, and perspective-shifting trips worth talking about."
    )
  ]

  private let environment: AppEnvironment

  private var results: [User] = []
  private var topics: [Topic] = SearchViewController.fallbackTopics
  private var recentSearches: [String] = []
  private var followingIDs = Set<String>()
  private var inFlightFollowUserIDs = Set<String>()
  private var searchTask: Task<Void, Never>?
  private var topicsTask: Task<Void, Never>?
  private var needsTableReloadOnAppear = false
  private var isShowingAllTopics = false
  private var isSearching = false
  private var searchFailureState: SearchFailureState?

  private lazy var clearSearchButton: UIButton = {
    let button = UIButton(type: .system)
    button.tintColor = Theme.secondaryText
    button.setImage(AppIcon.close.image, for: .normal)
    button.imageView?.contentMode = .scaleAspectFit
    button.addTarget(self, action: #selector(clearSearchTapped), for: .touchUpInside)
    button.accessibilityLabel = L10n.tr("common.clear")
    return button
  }()

  private lazy var searchField: UISearchTextField = {
    let field = UISearchTextField()
    field.translatesAutoresizingMaskIntoConstraints = false
    field.placeholder = Copy.searchPlaceholder
    field.borderStyle = .none
    field.autocorrectionType = .no
    field.autocapitalizationType = .none
    field.spellCheckingType = .no
    field.clearButtonMode = .never
    field.returnKeyType = .search
    field.textColor = Theme.text
    field.tintColor = Theme.text
    field.font = Theme.Typography.fieldValue
    field.adjustsFontForContentSizeCategory = true
    field.attributedPlaceholder = NSAttributedString(
      string: Copy.searchPlaceholder,
      attributes: Theme.Typography.placeholderAttributes()
    )
    field.leftView = makeSearchIconView()
    field.leftViewMode = .always
    field.rightView = makeDismissKeyboardAccessoryView()
    field.rightViewMode = .always
    field.delegate = self
    field.addTarget(self, action: #selector(searchTextChanged(_:)), for: .editingChanged)
    Theme.applyTextInputStyle(to: field, cornerRadius: 18)
    return field
  }()

  private let headerStack: UIStackView = {
    let stack = UIStackView()
    stack.axis = .vertical
    stack.spacing = 0
    stack.translatesAutoresizingMaskIntoConstraints = false
    return stack
  }()

  private let discoverScrollView: UIScrollView = {
    let scrollView = UIScrollView()
    scrollView.translatesAutoresizingMaskIntoConstraints = false
    scrollView.alwaysBounceVertical = true
    scrollView.keyboardDismissMode = .onDrag
    return scrollView
  }()

  private let discoverContentStack: UIStackView = {
    let stack = UIStackView()
    stack.axis = .vertical
    stack.spacing = Constants.sectionSpacing
    stack.translatesAutoresizingMaskIntoConstraints = false
    return stack
  }()

  private let topicsRowsStack: UIStackView = {
    let stack = UIStackView()
    stack.axis = .vertical
    stack.spacing = Constants.rowSpacing
    return stack
  }()

  private let recentRowsStack: UIStackView = {
    let stack = UIStackView()
    stack.axis = .vertical
    stack.spacing = Constants.rowSpacing
    return stack
  }()

  private lazy var topicsSeeAllButton: UIButton = {
    let button = UIButton(type: .system)
    button.setTitle(Copy.seeAll, for: .normal)
    button.setTitleColor(Theme.secondaryText, for: .normal)
    button.titleLabel?.font = Theme.Typography.secondaryAction
    button.titleLabel?.adjustsFontForContentSizeCategory = true
    button.contentHorizontalAlignment = .trailing
    button.addTarget(self, action: #selector(toggleTopicsExpansion), for: .touchUpInside)
    return button
  }()

  private lazy var clearAllButton: UIButton = {
    let button = UIButton(type: .system)
    button.setTitle(Copy.clearAll, for: .normal)
    button.setTitleColor(Theme.secondaryText, for: .normal)
    button.titleLabel?.font = Theme.Typography.secondaryAction
    button.titleLabel?.adjustsFontForContentSizeCategory = true
    button.contentHorizontalAlignment = .trailing
    button.addTarget(self, action: #selector(clearAllRecentSearchesTapped), for: .touchUpInside)
    return button
  }()

  private let recentEmptyLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 0
    label.text = Copy.recentEmpty
    return label
  }()

  private let tableView: UITableView = {
    let tableView = UITableView(frame: .zero, style: .plain)
    tableView.translatesAutoresizingMaskIntoConstraints = false
    tableView.separatorStyle = .none
    tableView.keyboardDismissMode = .onDrag
    tableView.backgroundColor = .clear
    return tableView
  }()

  private let emptyLabel: UILabel = {
    let label = UILabel()
    label.text = L10n.tr("search.empty")
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .center
    label.numberOfLines = 0
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()

  init(environment: AppEnvironment) {
    self.environment = environment
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  deinit {
    searchTask?.cancel()
    topicsTask?.cancel()
  }

  override func viewDidLoad() {
    super.viewDidLoad()

    configureNavigationTitle()

    recentSearches = loadRecentSearches()

    configureViewHierarchy()
    configureDiscoverSections()

    tableView.register(UserCell.self, forCellReuseIdentifier: UserCell.reuseID)
    tableView.dataSource = self
    tableView.delegate = self

    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleFollowRelationshipChange(_:)),
      name: .followRelationshipDidChange,
      object: nil
    )

    renderRecentSearches()
    renderTopics()
    renderState()
    loadTopics()
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
    updateBottomInsets()
  }

  private func configureViewHierarchy() {
    headerStack.addArrangedSubview(searchField)

    view.addSubview(headerStack)
    view.addSubview(discoverScrollView)
    view.addSubview(tableView)
    view.addSubview(emptyLabel)

    discoverScrollView.addSubview(discoverContentStack)

    tableView.contentInset = UIEdgeInsets(top: 0, left: 0, bottom: 32, right: 0)
    discoverScrollView.contentInset = UIEdgeInsets(top: 0, left: 0, bottom: 32, right: 0)

    NSLayoutConstraint.activate([
      searchField.heightAnchor.constraint(equalToConstant: Constants.searchFieldHeight),

      headerStack.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 12),
      headerStack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: Constants.horizontalInset),
      headerStack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -Constants.horizontalInset),

      discoverScrollView.topAnchor.constraint(equalTo: headerStack.bottomAnchor, constant: 18),
      discoverScrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      discoverScrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
      discoverScrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),

      discoverContentStack.topAnchor.constraint(equalTo: discoverScrollView.contentLayoutGuide.topAnchor),
      discoverContentStack.bottomAnchor.constraint(equalTo: discoverScrollView.contentLayoutGuide.bottomAnchor),
      discoverContentStack.leadingAnchor.constraint(
        equalTo: discoverScrollView.frameLayoutGuide.leadingAnchor,
        constant: Constants.horizontalInset
      ),
      discoverContentStack.trailingAnchor.constraint(
        equalTo: discoverScrollView.frameLayoutGuide.trailingAnchor,
        constant: -Constants.horizontalInset
      ),

      tableView.topAnchor.constraint(equalTo: headerStack.bottomAnchor, constant: 24),
      tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
      tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),

      emptyLabel.topAnchor.constraint(equalTo: headerStack.bottomAnchor, constant: 88),
      emptyLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: Constants.horizontalInset),
      emptyLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -Constants.horizontalInset)
    ])
  }

  private func configureNavigationTitle() {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.navigationTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = Copy.title
    navigationItem.titleView = titleLabel
    title = Copy.title
  }

  private func updateBottomInsets() {
    let bottomInset: CGFloat

    if navigationController?.viewControllers.first === self,
       let mainTabBarController = parentMainController {
      bottomInset = mainTabBarController.rootContentBottomInset
    } else {
      bottomInset = view.safeAreaInsets.bottom + 20
    }

    if tableView.contentInset.bottom != bottomInset {
      tableView.contentInset.bottom = bottomInset
      tableView.verticalScrollIndicatorInsets.bottom = bottomInset
    }

    if discoverScrollView.contentInset.bottom != bottomInset {
      discoverScrollView.contentInset.bottom = bottomInset
      discoverScrollView.verticalScrollIndicatorInsets.bottom = bottomInset
    }
  }

  private func configureDiscoverSections() {
    discoverContentStack.addArrangedSubview(makeTopicsSection())
    discoverContentStack.addArrangedSubview(makeRecentSearchesSection())
  }

  private func makeTopicsSection() -> UIView {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.bodyEditorialLarge
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = Copy.topicsTitle

    let spacer = UIView()
    let headerRow = UIStackView(arrangedSubviews: [titleLabel, spacer, topicsSeeAllButton])
    headerRow.axis = .horizontal
    headerRow.alignment = .firstBaseline

    let subtitleLabel = UILabel()
    subtitleLabel.font = Theme.Typography.helperText
    subtitleLabel.adjustsFontForContentSizeCategory = true
    subtitleLabel.textColor = Theme.secondaryText
    subtitleLabel.text = Copy.topicsSubtitle
    subtitleLabel.numberOfLines = 0

    let stack = UIStackView(arrangedSubviews: [
      headerRow,
      subtitleLabel,
      makeSpacer(height: 12),
      topicsRowsStack
    ])
    stack.axis = .vertical
    stack.spacing = 6
    return stack
  }

  private func makeRecentSearchesSection() -> UIView {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.bodyEditorialLarge
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = Copy.recentSearches

    let spacer = UIView()
    let headerRow = UIStackView(arrangedSubviews: [titleLabel, spacer, clearAllButton])
    headerRow.axis = .horizontal
    headerRow.alignment = .firstBaseline

    let stack = UIStackView(arrangedSubviews: [
      headerRow,
      makeSpacer(height: 10),
      recentEmptyLabel,
      recentRowsStack
    ])
    stack.axis = .vertical
    stack.spacing = 0
    return stack
  }

  private func loadTopics() {
    topicsTask?.cancel()
    topicsTask = Task { [weak self] in
      guard let self else { return }
      do {
        let fetchedTopics = try await environment.api.fetchTopics()
        guard !Task.isCancelled else { return }
        topics = fetchedTopics.isEmpty ? Self.fallbackTopics : fetchedTopics
      } catch {
        topics = Self.fallbackTopics
      }
      renderTopics()
    }
  }

  private func performSearch(query: String) async {
    let term = normalizedQuery(from: query)

    guard !term.isEmpty else {
      isSearching = false
      results = []
      followingIDs.removeAll()
      searchFailureState = nil
      renderState()
      return
    }

    do {
      let users = try await environment.api.searchUsers(query: term)
      isSearching = false
      results = users.filter { $0.id != environment.sessionStore.currentUser?.id }
      followingIDs = Set(results.filter(\.isFollowing).map(\.id))
      searchFailureState = nil
      renderState()
    } catch {
      isSearching = false
      results = []
      followingIDs.removeAll()
      searchFailureState = shouldPresentOfflineState(for: error) ? .offline : .generic
      renderState()
    }
  }

  private func scheduleSearch(query: String) {
    searchTask?.cancel()
    let term = normalizedQuery(from: query)

    guard !term.isEmpty else {
      isSearching = false
      results = []
      followingIDs.removeAll()
      searchFailureState = nil
      renderState()
      return
    }

    isSearching = true
    searchFailureState = nil
    renderState()

    searchTask = Task { [weak self] in
      guard let self else { return }
      do {
        try await Task.sleep(nanoseconds: 350_000_000)
      } catch {
        return
      }
      guard !Task.isCancelled else { return }
      await self.performSearch(query: term)
    }
  }

  private func renderState() {
    let isShowingSearchResults = !normalizedQuery(from: searchField.text ?? "").isEmpty
    discoverScrollView.isHidden = isShowingSearchResults
    tableView.isHidden = !isShowingSearchResults || results.isEmpty
    emptyLabel.isHidden = !isShowingSearchResults || isSearching || !results.isEmpty
    emptyLabel.text = searchFailureState?.message ?? L10n.tr("search.empty")
    tableView.reloadData()
  }

  private func renderTopics() {
    clearArrangedSubviews(in: topicsRowsStack)

    let visibleTopics = isShowingAllTopics ? topics : Array(topics.prefix(5))
    for (index, topic) in visibleTopics.enumerated() {
      let row = TopicRowButton()
      row.configure(
        topic: topic,
        imageURL: environment.api.resolveURL(path: topic.imageURL),
        showsSeparator: index < (visibleTopics.count - 1)
      )
      row.onTap = { [weak self] in
        self?.applySearchTerm(topic.name)
      }
      topicsRowsStack.addArrangedSubview(row)
    }

    topicsSeeAllButton.isHidden = topics.count <= 5
    topicsSeeAllButton.setTitle(isShowingAllTopics ? Copy.showLess : Copy.seeAll, for: .normal)
  }

  private func renderRecentSearches() {
    clearArrangedSubviews(in: recentRowsStack)

    recentEmptyLabel.isHidden = !recentSearches.isEmpty
    clearAllButton.isHidden = recentSearches.isEmpty

    for (index, term) in recentSearches.enumerated() {
      let row = RecentSearchRowView()
      row.configure(term: term, showsSeparator: index < (recentSearches.count - 1))
      row.onTap = { [weak self] in
        self?.applySearchTerm(term)
      }
      row.onRemove = { [weak self] in
        self?.removeRecentSearch(term)
      }
      recentRowsStack.addArrangedSubview(row)
    }
  }

  private func applySearchTerm(_ term: String) {
    let normalized = normalizedQuery(from: term)
    guard !normalized.isEmpty else { return }

    searchTask?.cancel()
    rememberRecentSearch(normalized)
    searchField.text = normalized
    updateClearSearchButtonState()
    searchField.resignFirstResponder()
    isSearching = true
    searchFailureState = nil
    renderState()

    Task { [weak self] in
      await self?.performSearch(query: normalized)
    }
  }

  private func rememberRecentSearch(_ term: String) {
    let normalized = normalizedQuery(from: term)
    guard !normalized.isEmpty else { return }

    recentSearches.removeAll { $0.caseInsensitiveCompare(normalized) == .orderedSame }
    recentSearches.insert(normalized, at: 0)
    if recentSearches.count > Constants.recentSearchLimit {
      recentSearches = Array(recentSearches.prefix(Constants.recentSearchLimit))
    }
    persistRecentSearches()
    renderRecentSearches()
  }

  private func removeRecentSearch(_ term: String) {
    recentSearches.removeAll { $0.caseInsensitiveCompare(term) == .orderedSame }
    persistRecentSearches()
    renderRecentSearches()
  }

  private func loadRecentSearches() -> [String] {
    let storedValues = UserDefaults.standard.stringArray(forKey: Constants.recentSearchesKey) ?? []
    return Array(storedValues.prefix(Constants.recentSearchLimit))
  }

  private func persistRecentSearches() {
    UserDefaults.standard.set(recentSearches, forKey: Constants.recentSearchesKey)
  }

  private func normalizedQuery(from value: String) -> String {
    value.trimmingCharacters(in: .whitespacesAndNewlines)
  }

  private func clearArrangedSubviews(in stackView: UIStackView) {
    let arrangedSubviews = stackView.arrangedSubviews
    arrangedSubviews.forEach { subview in
      stackView.removeArrangedSubview(subview)
      subview.removeFromSuperview()
    }
  }

  private func makeSearchIconView() -> UIView {
    let container = UIView(frame: CGRect(x: 0, y: 0, width: 42, height: 18))
    let imageView = UIImageView(image: AppIcon.search.image)
    imageView.tintColor = Theme.secondaryText
    imageView.contentMode = .scaleAspectFit
    imageView.frame = CGRect(x: 14, y: 1, width: 18, height: 18)
    container.addSubview(imageView)
    return container
  }

  private func makeDismissKeyboardAccessoryView() -> UIView {
    let container = UIView(frame: CGRect(x: 0, y: 0, width: 36, height: 18))
    clearSearchButton.frame = CGRect(x: 8, y: 0, width: 18, height: 18)
    container.addSubview(clearSearchButton)
    return container
  }

  private func updateClearSearchButtonState() {
    let hasText = !normalizedQuery(from: searchField.text ?? "").isEmpty
    clearSearchButton.alpha = hasText ? 1 : 0
    clearSearchButton.isUserInteractionEnabled = hasText
  }

  private func toggleFollow(userID: String) {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }
    guard !inFlightFollowUserIDs.contains(userID) else { return }
    guard userID != environment.sessionStore.currentUser?.id else { return }
    guard let row = results.firstIndex(where: { $0.id == userID }) else { return }
    let user = results[row]
    let isFollowing = followingIDs.contains(userID) || user.isFollowing
    let shouldFollow = !isFollowing
    inFlightFollowUserIDs.insert(userID)
    applyFollowState(shouldFollow, to: row)

    Task {
      do {
        if isFollowing {
          try await environment.api.unfollowUser(userID: userID)
        } else {
          try await environment.api.followUser(userID: userID)
        }
        inFlightFollowUserIDs.remove(userID)
        Haptics.shared.lightImpact()
      } catch {
        inFlightFollowUserIDs.remove(userID)
        guard let updatedRow = results.firstIndex(where: { $0.id == userID }) else { return }
        applyFollowState(isFollowing, to: updatedRow)
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(shouldFollow ? .follow : .unfollow)
      }
    }
  }

  private func applyFollowState(_ isFollowing: Bool, to row: Int) {
    let existingUser = results[row]
    let wasFollowing = followingIDs.contains(existingUser.id) || existingUser.isFollowing
    results[row] = existingUser.withFollowing(isFollowing)
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
        let updatedUser = results[row]
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

  private func makeSpacer(height: CGFloat) -> UIView {
    let spacer = UIView()
    spacer.translatesAutoresizingMaskIntoConstraints = false
    spacer.heightAnchor.constraint(equalToConstant: height).isActive = true
    return spacer
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

  @objc private func handleFollowRelationshipChange(_ notification: Notification) {
    guard let change = notification.followRelationshipChange else { return }
    guard let row = results.firstIndex(where: { $0.id == change.userID }) else { return }

    let updatedUser = results[row].withFollowing(change.isFollowing)
    guard updatedUser != results[row] || followingIDs.contains(change.userID) != change.isFollowing else { return }

    results[row] = updatedUser
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

  @objc private func clearSearchTapped() {
    searchTask?.cancel()
    searchField.text = ""
    isSearching = false
    results = []
    followingIDs.removeAll()
    searchFailureState = nil
    updateClearSearchButtonState()
    renderState()
  }

  @objc private func searchTextChanged(_ textField: UISearchTextField) {
    updateClearSearchButtonState()
    scheduleSearch(query: textField.text ?? "")
  }

  @objc private func toggleTopicsExpansion() {
    isShowingAllTopics.toggle()
    renderTopics()
  }

  @objc private func clearAllRecentSearchesTapped() {
    recentSearches.removeAll()
    persistRecentSearches()
    renderRecentSearches()
  }
}

extension SearchViewController: UITextFieldDelegate {
  func textFieldShouldReturn(_ textField: UITextField) -> Bool {
    let term = normalizedQuery(from: textField.text ?? "")
    textField.resignFirstResponder()
    searchTask?.cancel()

    guard !term.isEmpty else {
      isSearching = false
      results = []
      followingIDs.removeAll()
      searchFailureState = nil
      renderState()
      return true
    }

    rememberRecentSearch(term)
    isSearching = true
    searchFailureState = nil
    renderState()
    Task { [weak self] in
      await self?.performSearch(query: term)
    }
    return true
  }
}

extension SearchViewController: UITableViewDataSource {
  func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
    results.count
  }

  func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
    guard let cell = tableView.dequeueReusableCell(
      withIdentifier: UserCell.reuseID,
      for: indexPath
    ) as? UserCell else {
      return UITableViewCell()
    }

    let user = results[indexPath.row]
    let isFollowing = followingIDs.contains(user.id) || user.isFollowing
    let followEnabled = user.id != environment.sessionStore.currentUser?.id
    cell.configure(user: user, isFollowing: isFollowing, followEnabled: followEnabled)
    cell.onFollowTap = { [weak self] in
      self?.toggleFollow(userID: user.id)
    }
    return cell
  }
}

extension SearchViewController: UITableViewDelegate {
  func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
    tableView.deselectRow(at: indexPath, animated: true)
    let user = results[indexPath.row]
    let profileVC = UserProfileViewController(environment: environment, userID: user.id, initialUser: user)
    navigationController?.pushViewController(profileVC, animated: true)
  }
}

@MainActor
private final class TopicRowButton: UIControl {
  var onTap: (() -> Void)?

  private let artworkView = TopicArtworkView()
  private let titleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.bodyEditorialMedium
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 1
    return label
  }()

  private let descriptionLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 2
    return label
  }()

  private let separatorView: UIView = {
    let view = UIView()
    view.backgroundColor = Theme.separator
    view.translatesAutoresizingMaskIntoConstraints = false
    return view
  }()

  override init(frame: CGRect) {
    super.init(frame: frame)
    translatesAutoresizingMaskIntoConstraints = false
    addTarget(self, action: #selector(handleTap), for: .touchUpInside)

    let labelsStack = UIStackView(arrangedSubviews: [titleLabel, descriptionLabel])
    labelsStack.axis = .vertical
    labelsStack.alignment = .leading
    labelsStack.spacing = 4

    let chevronView = UIImageView(image: AppIcon.chevronRight.image)
    chevronView.tintColor = Theme.secondaryText
    chevronView.contentMode = .scaleAspectFit
    chevronView.translatesAutoresizingMaskIntoConstraints = false

    let spacer = UIView()
    let row = UIStackView(arrangedSubviews: [artworkView, labelsStack, spacer, chevronView])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = 14
    row.translatesAutoresizingMaskIntoConstraints = false
    row.isUserInteractionEnabled = false

    addSubview(row)
    addSubview(separatorView)

    NSLayoutConstraint.activate([
      artworkView.widthAnchor.constraint(equalToConstant: 52),
      artworkView.heightAnchor.constraint(equalToConstant: 52),
      chevronView.widthAnchor.constraint(equalToConstant: 28),
      chevronView.heightAnchor.constraint(equalToConstant: 28),

      row.topAnchor.constraint(equalTo: topAnchor, constant: 14),
      row.leadingAnchor.constraint(equalTo: leadingAnchor),
      row.trailingAnchor.constraint(equalTo: trailingAnchor),
      row.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -14),

      separatorView.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale),
      separatorView.leadingAnchor.constraint(equalTo: leadingAnchor),
      separatorView.trailingAnchor.constraint(equalTo: trailingAnchor),
      separatorView.bottomAnchor.constraint(equalTo: bottomAnchor)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  func configure(topic: Topic, imageURL: URL?, showsSeparator: Bool) {
    titleLabel.text = topic.name
    descriptionLabel.text = topic.description
    separatorView.isHidden = !showsSeparator
    artworkView.configure(title: topic.name, imageURL: imageURL)
    accessibilityLabel = topic.name
  }

  @objc private func handleTap() {
    onTap?()
  }
}

@MainActor
private final class RecentSearchRowView: UIView {
  var onTap: (() -> Void)?
  var onRemove: (() -> Void)?

  private let rowButton: UIButton = {
    let button = UIButton(type: .custom)
    button.translatesAutoresizingMaskIntoConstraints = false
    button.contentHorizontalAlignment = .leading
    return button
  }()

  private let titleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.bodyEditorialSmall
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 1
    return label
  }()

  private let removeButton: UIButton = {
    let button = UIButton(type: .system)
    button.translatesAutoresizingMaskIntoConstraints = false
    button.tintColor = Theme.secondaryText
    button.setImage(AppIcon.close.image, for: .normal)
    button.accessibilityLabel = L10n.tr("common.clear")
    return button
  }()

  private let separatorView: UIView = {
    let view = UIView()
    view.backgroundColor = Theme.separator
    view.translatesAutoresizingMaskIntoConstraints = false
    return view
  }()

  override init(frame: CGRect) {
    super.init(frame: frame)
    translatesAutoresizingMaskIntoConstraints = false

    rowButton.addTarget(self, action: #selector(handleTap), for: .touchUpInside)
    removeButton.addTarget(self, action: #selector(handleRemove), for: .touchUpInside)

    let iconView = UIImageView(image: UIImage(systemName: "clock"))
    iconView.tintColor = Theme.secondaryText
    iconView.contentMode = .scaleAspectFit
    iconView.translatesAutoresizingMaskIntoConstraints = false

    let row = UIStackView(arrangedSubviews: [iconView, titleLabel])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = 14
    row.isUserInteractionEnabled = false
    row.translatesAutoresizingMaskIntoConstraints = false

    rowButton.addSubview(row)
    addSubview(rowButton)
    addSubview(removeButton)
    addSubview(separatorView)

    NSLayoutConstraint.activate([
      iconView.widthAnchor.constraint(equalToConstant: 18),
      iconView.heightAnchor.constraint(equalToConstant: 18),

      rowButton.topAnchor.constraint(equalTo: topAnchor, constant: 10),
      rowButton.leadingAnchor.constraint(equalTo: leadingAnchor),
      rowButton.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -10),

      removeButton.leadingAnchor.constraint(equalTo: rowButton.trailingAnchor, constant: 12),
      removeButton.trailingAnchor.constraint(equalTo: trailingAnchor),
      removeButton.centerYAnchor.constraint(equalTo: rowButton.centerYAnchor),
      removeButton.widthAnchor.constraint(equalToConstant: 32),
      removeButton.heightAnchor.constraint(equalToConstant: 32),

      row.topAnchor.constraint(equalTo: rowButton.topAnchor),
      row.leadingAnchor.constraint(equalTo: rowButton.leadingAnchor),
      row.trailingAnchor.constraint(equalTo: rowButton.trailingAnchor),
      row.bottomAnchor.constraint(equalTo: rowButton.bottomAnchor),

      separatorView.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale),
      separatorView.leadingAnchor.constraint(equalTo: leadingAnchor),
      separatorView.trailingAnchor.constraint(equalTo: trailingAnchor),
      separatorView.bottomAnchor.constraint(equalTo: bottomAnchor)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  func configure(term: String, showsSeparator: Bool) {
    titleLabel.text = term
    separatorView.isHidden = !showsSeparator
    accessibilityLabel = term
  }

  @objc private func handleTap() {
    onTap?()
  }

  @objc private func handleRemove() {
    onRemove?()
  }
}

@MainActor
private final class TopicArtworkView: UIView {
  private static let imageCache = NSCache<NSString, UIImage>()

  private let imageView: UIImageView = {
    let imageView = UIImageView()
    imageView.translatesAutoresizingMaskIntoConstraints = false
    imageView.contentMode = .scaleAspectFill
    imageView.clipsToBounds = true
    return imageView
  }()

  private let initialLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.sectionHeading
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.textAlignment = .center
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()

  private var imageTask: Task<Void, Never>?
  private var currentURLString: String?

  override init(frame: CGRect) {
    super.init(frame: frame)
    translatesAutoresizingMaskIntoConstraints = false
    clipsToBounds = true
    Theme.applySurface(to: self, cornerRadius: 0, fillColor: Theme.elevatedSurfaceBackground)

    addSubview(imageView)
    addSubview(initialLabel)

    NSLayoutConstraint.activate([
      imageView.topAnchor.constraint(equalTo: topAnchor),
      imageView.bottomAnchor.constraint(equalTo: bottomAnchor),
      imageView.leadingAnchor.constraint(equalTo: leadingAnchor),
      imageView.trailingAnchor.constraint(equalTo: trailingAnchor),

      initialLabel.centerXAnchor.constraint(equalTo: centerXAnchor),
      initialLabel.centerYAnchor.constraint(equalTo: centerYAnchor)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  deinit {
    imageTask?.cancel()
  }

  override func layoutSubviews() {
    super.layoutSubviews()
    let diameter = min(bounds.width, bounds.height)
    Theme.applySurface(to: self, cornerRadius: diameter / 2, fillColor: Theme.elevatedSurfaceBackground)
  }

  func configure(title: String, imageURL: URL?) {
    imageTask?.cancel()
    currentURLString = imageURL?.absoluteString
    initialLabel.text = String(title.prefix(1)).uppercased()
    initialLabel.isHidden = false
    imageView.image = nil

    guard let imageURL else { return }

    let cacheKey = imageURL.absoluteString as NSString
    if let cachedImage = Self.imageCache.object(forKey: cacheKey) {
      show(image: cachedImage)
      return
    }

    imageTask = Task { [weak self] in
      guard let self else { return }
      do {
        let (data, response) = try await URLSession.shared.data(from: imageURL)
        guard let httpResponse = response as? HTTPURLResponse,
              (200 ... 299).contains(httpResponse.statusCode),
              let image = UIImage(data: data) else {
          return
        }

        Self.imageCache.setObject(image, forKey: cacheKey)
        guard !Task.isCancelled else { return }
        guard self.currentURLString == imageURL.absoluteString else { return }
        self.show(image: image)
      } catch {
        // Keep the initial placeholder when artwork is unavailable.
      }
    }
  }

  private func show(image: UIImage) {
    imageView.image = image
    initialLabel.isHidden = true
  }
}
