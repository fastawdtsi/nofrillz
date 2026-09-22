import UIKit

@MainActor
final class MainTabBarController: UIViewController {
  private enum Constants {
    static let barButtonHeight: CGFloat = 60
    static let barContentHeight: CGFloat = 64
    static let contentBottomSpacing: CGFloat = 12
    static let hiddenBottomPadding: CGFloat = 16
    static let barIconSize: CGFloat = 28
    static let barLabelSpacing: CGFloat = 4
  }

  private let environment: AppEnvironment
  private var bottomBarHiddenByScroll = false
  private let bottomBarView = UIView()
  private let buttonStackView = UIStackView()
  private var bottomBarHeightConstraint: NSLayoutConstraint?

  private lazy var feedViewController: FeedViewController = {
    let viewController = FeedViewController(environment: environment)
    viewController.onFloatingControlsVisibilityChange = { [weak self] hidden, animated in
      self?.setBottomBarHiddenByScroll(hidden, animated: animated)
    }
    return viewController
  }()

  private lazy var feedNavigationController = NoFrillzNavigationController(rootViewController: feedViewController)

  private lazy var searchButton = makeBarButton(
    image: AppIcon.discover.image,
    title: L10n.tr("tab.search"),
    accessibilityLabel: L10n.tr("tab.search"),
    action: #selector(searchTapped)
  )

  private lazy var postButton = makeBarButton(
    image: AppIcon.pencil.image,
    title: L10n.tr("create.write_title"),
    accessibilityLabel: L10n.tr("create.write_title"),
    action: #selector(postTapped)
  )

  private lazy var profileButton = makeBarButton(
    image: AppIcon.profile.image,
    title: L10n.tr("tab.profile"),
    accessibilityLabel: L10n.tr("tab.profile"),
    action: #selector(profileTapped)
  )

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
    view.backgroundColor = Theme.background
    configureNavigationBarAppearance(for: feedNavigationController)
    feedNavigationController.delegate = self
    embedFeedNavigationController()
    configureBottomBar()
  }

  override func viewDidLayoutSubviews() {
    super.viewDidLayoutSubviews()
    updateBottomBarMetrics()
    applyBottomBarVisibility(animated: false)
    feedNavigationController.view.setNeedsLayout()
  }

  var rootContentBottomInset: CGFloat {
    view.safeAreaInsets.bottom + Constants.barContentHeight + Constants.contentBottomSpacing
  }

  var feedbackToastBottomInset: CGFloat {
    if isBottomBarVisible {
      return rootContentBottomInset + 12
    }
    return view.safeAreaInsets.bottom + 16
  }

  private func embedFeedNavigationController() {
    addChild(feedNavigationController)
    feedNavigationController.view.translatesAutoresizingMaskIntoConstraints = false
    view.addSubview(feedNavigationController.view)
    NSLayoutConstraint.activate([
      feedNavigationController.view.topAnchor.constraint(equalTo: view.topAnchor),
      feedNavigationController.view.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      feedNavigationController.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      feedNavigationController.view.trailingAnchor.constraint(equalTo: view.trailingAnchor)
    ])
    feedNavigationController.didMove(toParent: self)
  }

  private func configureBottomBar() {
    bottomBarView.translatesAutoresizingMaskIntoConstraints = false
    bottomBarView.backgroundColor = Theme.background
    view.addSubview(bottomBarView)

    buttonStackView.translatesAutoresizingMaskIntoConstraints = false
    buttonStackView.axis = .horizontal
    buttonStackView.alignment = .fill
    buttonStackView.distribution = .fillEqually
    bottomBarView.addSubview(buttonStackView)

    [searchButton, postButton, profileButton].forEach { button in
      buttonStackView.addArrangedSubview(button)
      NSLayoutConstraint.activate([
        button.heightAnchor.constraint(equalToConstant: Constants.barButtonHeight)
      ])
    }

    bottomBarHeightConstraint = bottomBarView.heightAnchor.constraint(equalToConstant: Constants.barContentHeight)

    NSLayoutConstraint.activate([
      bottomBarView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      bottomBarView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
      bottomBarView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      bottomBarHeightConstraint!,

      buttonStackView.topAnchor.constraint(equalTo: bottomBarView.topAnchor),
      buttonStackView.leadingAnchor.constraint(equalTo: bottomBarView.leadingAnchor),
      buttonStackView.trailingAnchor.constraint(equalTo: bottomBarView.trailingAnchor),
      buttonStackView.heightAnchor.constraint(equalToConstant: Constants.barContentHeight)
    ])

    applyBottomBarVisibility(animated: false)
  }

  private func updateBottomBarMetrics() {
    bottomBarHeightConstraint?.constant = Constants.barContentHeight + view.safeAreaInsets.bottom
  }

  private func makeBarButton(
    image: UIImage?,
    title: String,
    accessibilityLabel: String,
    action: Selector
  ) -> UIButton {
    let button = UIButton(type: .custom)
    button.translatesAutoresizingMaskIntoConstraints = false
    button.backgroundColor = .clear
    button.accessibilityLabel = accessibilityLabel
    button.accessibilityTraits = .button
    button.addTarget(self, action: action, for: .touchUpInside)

    let stackView = UIStackView()
    stackView.translatesAutoresizingMaskIntoConstraints = false
    stackView.axis = .vertical
    stackView.alignment = .center
    stackView.spacing = Constants.barLabelSpacing
    stackView.isUserInteractionEnabled = false
    button.addSubview(stackView)

    let imageView = UIImageView(image: image?.withRenderingMode(.alwaysTemplate))
    imageView.translatesAutoresizingMaskIntoConstraints = false
    imageView.tintColor = Theme.text
    imageView.contentMode = .scaleAspectFit
    stackView.addArrangedSubview(imageView)

    let label = UILabel()
    label.text = title
    label.font = Theme.Typography.tabLabel
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .center
    stackView.addArrangedSubview(label)

    NSLayoutConstraint.activate([
      stackView.centerXAnchor.constraint(equalTo: button.centerXAnchor),
      stackView.centerYAnchor.constraint(equalTo: button.centerYAnchor),
      stackView.topAnchor.constraint(greaterThanOrEqualTo: button.topAnchor, constant: 6),
      stackView.bottomAnchor.constraint(lessThanOrEqualTo: button.bottomAnchor, constant: -6),

      imageView.widthAnchor.constraint(equalToConstant: Constants.barIconSize),
      imageView.heightAnchor.constraint(equalToConstant: Constants.barIconSize)
    ])

    return button
  }

  private func setBottomBarHiddenByScroll(_ hidden: Bool, animated: Bool) {
    guard bottomBarHiddenByScroll != hidden else { return }
    bottomBarHiddenByScroll = hidden
    applyBottomBarVisibility(animated: animated)
  }

  private func applyBottomBarVisibility(animated: Bool) {
    let shouldHide = !isBottomBarVisible
    let hiddenOffset = (bottomBarHeightConstraint?.constant ?? bottomBarView.bounds.height) + Constants.hiddenBottomPadding

    let updates = {
      self.bottomBarView.transform = shouldHide
        ? CGAffineTransform(translationX: 0, y: hiddenOffset)
        : .identity
      self.bottomBarView.alpha = shouldHide ? 0 : 1
      self.bottomBarView.isUserInteractionEnabled = !shouldHide
    }

    if animated {
      UIView.animate(
        withDuration: 0.24,
        delay: 0,
        options: [.curveEaseInOut, .beginFromCurrentState]
      ) {
        updates()
      }
    } else {
      updates()
    }
  }

  private var isBottomBarVisible: Bool {
    feedNavigationController.viewControllers.count == 1 && !bottomBarHiddenByScroll
  }

  @objc private func searchTapped() {
    feedNavigationController.pushViewController(
      SearchViewController(environment: environment),
      animated: true
    )
  }

  @objc private func profileTapped() {
    let profileViewController = UserProfileViewController(
      environment: environment,
      userID: environment.sessionStore.currentUser?.id ?? "",
      initialUser: environment.sessionStore.currentUser,
      showsLogoutButton: true,
      useAuthenticatedUserEndpoint: true
    )
    feedNavigationController.pushViewController(profileViewController, animated: true)
  }

  @objc private func postTapped() {
    let createPostViewController = CreatePostViewController(environment: environment)
    let navigationController = NoFrillzNavigationController(rootViewController: createPostViewController)

    createPostViewController.onPostCreated = { [weak navigationController] in
      navigationController?.dismiss(animated: true)
    }
    configureNavigationBarAppearance(for: navigationController)
    navigationController.modalPresentationStyle = .fullScreen
    navigationController.modalTransitionStyle = .coverVertical

    present(navigationController, animated: true)
  }

  func openNotificationPath(_ path: String) -> Bool {
    let trimmedPath = path.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmedPath.isEmpty else { return false }

    let normalizedPath: String
    if let url = URL(string: trimmedPath), url.scheme != nil {
      normalizedPath = url.path
    } else {
      normalizedPath = trimmedPath
    }

    let pathComponents = normalizedPath.split(separator: "/").map(String.init)
    guard pathComponents.count >= 2 else { return false }

    switch pathComponents[0] {
    case "posts":
      let postID = pathComponents[1]
      presentNotificationDestination {
        Task { @MainActor [weak self] in
          guard let self else { return }
          do {
            let post = try await self.environment.api.fetchPost(id: postID)
            self.feedNavigationController.pushViewController(
              PostDetailViewController(environment: self.environment, post: post),
              animated: true
            )
          } catch {
#if DEBUG
            print("[NoFrillzDebug][Push] Failed to open post \(postID): \(error)")
#endif
          }
        }
      }
      return true

    case "users":
      let userID = pathComponents[1]
      presentNotificationDestination { [weak self] in
        guard let self else { return }
        self.feedNavigationController.pushViewController(
          UserProfileViewController(environment: self.environment, userID: userID),
          animated: true
        )
      }
      return true

    default:
      return false
    }
  }

  private func presentNotificationDestination(_ action: @escaping () -> Void) {
    if let presentedViewController {
      presentedViewController.dismiss(animated: true) {
        action()
      }
    } else {
      action()
    }
  }

  private func configureNavigationBarAppearance(for navigationController: UINavigationController) {
    let appearance = UINavigationBarAppearance()
    Theme.configureNavigationBarAppearance(appearance)
    navigationController.navigationBar.standardAppearance = appearance
    navigationController.navigationBar.scrollEdgeAppearance = appearance
    navigationController.navigationBar.compactAppearance = appearance
    navigationController.navigationBar.tintColor = Theme.text
    navigationController.navigationBar.isTranslucent = true
  }
}

extension MainTabBarController: UINavigationControllerDelegate {
  func navigationController(
    _ navigationController: UINavigationController,
    didShow viewController: UIViewController,
    animated: Bool
  ) {
    applyBottomBarVisibility(animated: true)
  }
}
