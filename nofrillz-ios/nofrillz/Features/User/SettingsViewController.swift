import UIKit

final class SettingsViewController: BaseViewController {
  private let horizontalContentInset: CGFloat = 16
  private let environment: AppEnvironment

  private let scrollView: UIScrollView = {
    let scrollView = UIScrollView()
    scrollView.translatesAutoresizingMaskIntoConstraints = false
    scrollView.alwaysBounceVertical = true
    return scrollView
  }()

  private let contentStack: UIStackView = {
    let stack = UIStackView()
    stack.axis = .vertical
    stack.spacing = 0
    stack.translatesAutoresizingMaskIntoConstraints = false
    return stack
  }()

  private let avatarView = AvatarView()

  private let nameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.longFormHeadline
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 1
    return label
  }()

  private let usernameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.username
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 1
    return label
  }()

  private let emailLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.metadata
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 1
    return label
  }()

  private let statsLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.metadata
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 0
    return label
  }()

  private lazy var profileButton = makeRowButton(
    title: L10n.tr("tab.profile"),
    subtitle: L10n.tr("settings.profile_subtitle")
  )

  private lazy var notificationsButton = makeRowButton(
    title: L10n.tr("notifications.title"),
    subtitle: L10n.tr("settings.notifications_subtitle")
  )

  private lazy var securityButton = makeRowButton(
    title: L10n.tr("security.title"),
    subtitle: L10n.tr("settings.security_subtitle")
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
    configureNavigationTitle()
    setupView()
    wireActions()
    refreshSummary()
  }

  override func viewWillAppear(_ animated: Bool) {
    super.viewWillAppear(animated)
    refreshSummary()
  }

  private func configureNavigationTitle() {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.navigationTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = L10n.tr("settings.title")
    navigationItem.titleView = titleLabel
  }

  private func setupView() {
    view.addSubview(scrollView)
    scrollView.addSubview(contentStack)

    NSLayoutConstraint.activate([
      scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),

      contentStack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 20),
      contentStack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -28),
      contentStack.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor),
      contentStack.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor)
    ])

    contentStack.addArrangedSubview(makeSummarySection())
    contentStack.addArrangedSubview(makeSpacer(height: 24))
    contentStack.addArrangedSubview(makeOptionsSection())
  }

  private func makeSummarySection() -> UIView {
    let container = UIView()
    container.translatesAutoresizingMaskIntoConstraints = false

    avatarView.widthAnchor.constraint(equalToConstant: 72).isActive = true
    avatarView.heightAnchor.constraint(equalToConstant: 72).isActive = true

    let identityStack = UIStackView(arrangedSubviews: [nameLabel, usernameLabel, emailLabel, statsLabel])
    identityStack.axis = .vertical
    identityStack.alignment = .leading
    identityStack.spacing = 4

    let headerRow = UIStackView(arrangedSubviews: [avatarView, identityStack])
    headerRow.axis = .horizontal
    headerRow.alignment = .center
    headerRow.spacing = 14
    headerRow.translatesAutoresizingMaskIntoConstraints = false

    let separator = makeSeparator()
    container.addSubview(headerRow)
    container.addSubview(separator)

    NSLayoutConstraint.activate([
      headerRow.topAnchor.constraint(equalTo: container.topAnchor),
      headerRow.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      headerRow.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),

      separator.topAnchor.constraint(equalTo: headerRow.bottomAnchor, constant: 20),
      separator.leadingAnchor.constraint(equalTo: container.leadingAnchor),
      separator.trailingAnchor.constraint(equalTo: container.trailingAnchor),
      separator.bottomAnchor.constraint(equalTo: container.bottomAnchor)
    ])

    return container
  }

  private func makeOptionsSection() -> UIView {
    let container = UIView()
    container.translatesAutoresizingMaskIntoConstraints = false

    let stack = UIStackView(arrangedSubviews: [
      profileButton,
      makeSeparator(),
      notificationsButton,
      makeSeparator(),
      securityButton,
      makeSeparator()
    ])
    stack.axis = .vertical
    stack.alignment = .fill
    stack.spacing = 0
    stack.translatesAutoresizingMaskIntoConstraints = false

    container.addSubview(stack)

    NSLayoutConstraint.activate([
      stack.topAnchor.constraint(equalTo: container.topAnchor),
      stack.bottomAnchor.constraint(equalTo: container.bottomAnchor),
      stack.leadingAnchor.constraint(equalTo: container.leadingAnchor),
      stack.trailingAnchor.constraint(equalTo: container.trailingAnchor)
    ])

    return container
  }

  private func makeRowButton(title: String, subtitle: String?) -> UIButton {
    let button = UIButton(type: .custom)
    button.contentHorizontalAlignment = .fill
    button.backgroundColor = .clear
    button.translatesAutoresizingMaskIntoConstraints = false
    button.heightAnchor.constraint(greaterThanOrEqualToConstant: 72).isActive = true

    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.settingsRowTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = title
    titleLabel.numberOfLines = 1

    let subtitleLabel = UILabel()
    subtitleLabel.font = Theme.Typography.settingsDescription
    subtitleLabel.adjustsFontForContentSizeCategory = true
    subtitleLabel.textColor = Theme.secondaryText
    subtitleLabel.text = subtitle
    subtitleLabel.numberOfLines = 1

    let labelsStack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel])
    labelsStack.axis = .vertical
    labelsStack.alignment = .leading
    labelsStack.spacing = 3
    labelsStack.isUserInteractionEnabled = false

    let chevronView = UIImageView(image: AppIcon.chevronRight.image)
    chevronView.tintColor = Theme.secondaryText
    chevronView.contentMode = .scaleAspectFit
    chevronView.translatesAutoresizingMaskIntoConstraints = false
    NSLayoutConstraint.activate([
      chevronView.widthAnchor.constraint(equalToConstant: 32),
      chevronView.heightAnchor.constraint(equalToConstant: 32)
    ])

    let row = UIStackView(arrangedSubviews: [labelsStack, chevronView])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = 12
    row.isUserInteractionEnabled = false
    row.translatesAutoresizingMaskIntoConstraints = false

    button.addSubview(row)

    NSLayoutConstraint.activate([
      row.topAnchor.constraint(equalTo: button.topAnchor, constant: 18),
      row.bottomAnchor.constraint(equalTo: button.bottomAnchor, constant: -18),
      row.leadingAnchor.constraint(equalTo: button.leadingAnchor, constant: horizontalContentInset),
      row.trailingAnchor.constraint(equalTo: button.trailingAnchor, constant: -horizontalContentInset)
    ])

    return button
  }

  private func makeSeparator() -> UIView {
    let view = UIView()
    view.backgroundColor = Theme.separator
    view.translatesAutoresizingMaskIntoConstraints = false
    view.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
    return view
  }

  private func makeSpacer(height: CGFloat) -> UIView {
    let view = UIView()
    view.translatesAutoresizingMaskIntoConstraints = false
    view.heightAnchor.constraint(equalToConstant: height).isActive = true
    return view
  }

  private func wireActions() {
    profileButton.addTarget(self, action: #selector(profileTapped), for: .touchUpInside)
    notificationsButton.addTarget(self, action: #selector(notificationsTapped), for: .touchUpInside)
    securityButton.addTarget(self, action: #selector(securityTapped), for: .touchUpInside)
  }

  private func refreshSummary() {
    let user = environment.sessionStore.currentUser
    let username = user?.username ?? L10n.tr("profile.default_user_handle")
    nameLabel.text = user?.displayName ?? L10n.tr("auth.brand")
    usernameLabel.text = "@\(username)"
    let email = user?.email?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    emailLabel.text = email.isEmpty ? L10n.tr("settings.no_email") : email

    if let user {
      statsLabel.text = L10n.tr("settings.profile_stats", user.postCount, user.followerCount, user.followingCount)
    } else {
      statsLabel.text = nil
    }

    avatarView.configure(
      username: username,
      firstName: user?.firstName,
      lastName: user?.lastName,
      avatarURL: user?.avatarURL
    )
  }

  @objc private func profileTapped() {
    let viewController = ProfileSettingsViewController(environment: environment)
    navigationController?.pushViewController(viewController, animated: true)
  }

  @objc private func notificationsTapped() {
    let viewController = NotificationsSettingsViewController()
    navigationController?.pushViewController(viewController, animated: true)
  }

  @objc private func securityTapped() {
    let viewController = SecuritySettingsViewController(environment: environment)
    navigationController?.pushViewController(viewController, animated: true)
  }
}
