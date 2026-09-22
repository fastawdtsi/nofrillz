import UIKit

final class ProfileSettingsViewController: BaseViewController {
  private let horizontalContentInset: CGFloat = 16
  private let sectionTopInset: CGFloat = 20
  private let sectionBottomInset: CGFloat = 22

  private struct EditableProfile: Equatable {
    let displayName: String
    let email: String
    let about: String
  }

  private enum Copy {
    static let title = L10n.tr("profile.edit.title")
    static let done = L10n.tr("profile.edit.done")
    static let displayName = L10n.tr("profile.edit.display_name")
    static let bio = L10n.tr("profile.edit.bio")
    static let saveFailed = L10n.tr("profile.edit.save_failed")
    static let invalidDisplayName = L10n.tr("profile.edit.display_name_invalid")
    static let missingProfileFields = L10n.tr("profile.edit.missing_fields")
  }

  private let environment: AppEnvironment
  private let maxAboutLength = 100

  private var originalProfile: EditableProfile?
  private var currentUser: User?
  private var isSavingProfile = false

  private let scrollView: UIScrollView = {
    let scrollView = UIScrollView()
    scrollView.translatesAutoresizingMaskIntoConstraints = false
    scrollView.alwaysBounceVertical = true
    scrollView.keyboardDismissMode = .interactive
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
    label.font = Theme.Typography.displayName
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 2
    return label
  }()

  private let displayNameField = NFTextField(placeholder: Copy.displayName)

  private let usernameValueLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.username
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 1
    return label
  }()

  private let aboutField: UITextView = {
    let textView = UITextView()
    textView.font = Theme.Typography.bodyEditorialMedium
    textView.adjustsFontForContentSizeCategory = true
    textView.textColor = Theme.text
    textView.backgroundColor = .clear
    textView.textContainerInset = .zero
    textView.textContainer.lineFragmentPadding = 0
    textView.showsVerticalScrollIndicator = false
    textView.translatesAutoresizingMaskIntoConstraints = false
    return textView
  }()

  private let aboutCountLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.characterCounter
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText.withAlphaComponent(0.7)
    label.textAlignment = .right
    label.text = L10n.tr("common.count_of_total", 0, 100)
    return label
  }()

  private lazy var saveBarButtonItem: UIBarButtonItem = {
    makeNavigationTextBarButtonItem(
      title: Copy.done,
      accessibilityLabel: Copy.done,
      action: #selector(saveProfileTapped)
    )
  }()

  private let activityIndicator: UIActivityIndicatorView = {
    let indicator = UIActivityIndicatorView(style: .medium)
    indicator.color = Theme.text
    indicator.hidesWhenStopped = true
    indicator.translatesAutoresizingMaskIntoConstraints = false
    return indicator
  }()

  private lazy var activityIndicatorContainer: UIView = {
    let container = UIView(frame: CGRect(x: 0, y: 0, width: 44, height: 44))
    container.backgroundColor = .clear
    container.addSubview(activityIndicator)
    NSLayoutConstraint.activate([
      activityIndicator.centerXAnchor.constraint(equalTo: container.centerXAnchor),
      activityIndicator.centerYAnchor.constraint(equalTo: container.centerYAnchor)
    ])
    return container
  }()

  private lazy var activityIndicatorBarButtonItem: UIBarButtonItem = {
    let item = UIBarButtonItem(customView: activityIndicatorContainer)
    Theme.applyEditorialBarButtonStyle(to: item)
    return item
  }()

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
    configureForm()
    setupView()
    apply(user: environment.sessionStore.currentUser)
    Task { await refreshCurrentUser() }
  }

  private func configureNavigationTitle() {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.navigationTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = Copy.title
    navigationItem.titleView = titleLabel
    navigationItem.rightBarButtonItem = saveBarButtonItem
  }

  private func configureForm() {
    displayNameField.autocapitalizationType = .words
    displayNameField.textContentType = .name
    displayNameField.returnKeyType = .done
    aboutField.delegate = self

    configureEditableFieldAppearance(displayNameField)
    displayNameField.addTarget(self, action: #selector(profileFieldChanged), for: .editingChanged)
    displayNameField.heightAnchor.constraint(equalToConstant: 44).isActive = true
    aboutField.heightAnchor.constraint(equalToConstant: 132).isActive = true
  }

  private func setupView() {
    view.addSubview(scrollView)
    scrollView.addSubview(contentStack)

    NSLayoutConstraint.activate([
      scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),

      contentStack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 12),
      contentStack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -28),
      contentStack.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor),
      contentStack.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor)
    ])

    contentStack.addArrangedSubview(makeSummarySection())
    contentStack.addArrangedSubview(makeSpacer(height: 18))
    contentStack.addArrangedSubview(makeFormSection())
  }

  private func makeSummarySection() -> UIView {
    let container = UIView()
    container.translatesAutoresizingMaskIntoConstraints = false

    avatarView.widthAnchor.constraint(equalToConstant: 96).isActive = true
    avatarView.heightAnchor.constraint(equalToConstant: 96).isActive = true

    let row = UIStackView(arrangedSubviews: [avatarView, nameLabel])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = 22
    row.translatesAutoresizingMaskIntoConstraints = false

    let separator = makeSeparator()
    container.addSubview(row)
    container.addSubview(separator)

    NSLayoutConstraint.activate([
      row.topAnchor.constraint(equalTo: container.topAnchor, constant: 8),
      row.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      row.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),

      separator.topAnchor.constraint(equalTo: row.bottomAnchor, constant: 28),
      separator.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      separator.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),
      separator.bottomAnchor.constraint(equalTo: container.bottomAnchor)
    ])

    return container
  }

  private func makeFormSection() -> UIView {
    let content = UIStackView(arrangedSubviews: [
      makeStackedSection(title: Copy.displayName, contentView: displayNameField),
      makeStackedSection(title: L10n.tr("field.username"), contentView: usernameValueLabel),
      makeAboutSection()
    ])
    content.axis = .vertical
    content.spacing = 0
    content.translatesAutoresizingMaskIntoConstraints = false

    let container = UIView()
    container.translatesAutoresizingMaskIntoConstraints = false
    container.addSubview(content)

    NSLayoutConstraint.activate([
      content.topAnchor.constraint(equalTo: container.topAnchor),
      content.bottomAnchor.constraint(equalTo: container.bottomAnchor),
      content.leadingAnchor.constraint(equalTo: container.leadingAnchor),
      content.trailingAnchor.constraint(equalTo: container.trailingAnchor)
    ])

    return container
  }

  private func configureEditableFieldAppearance(_ field: NFTextField) {
    field.backgroundColor = .clear
    field.layer.cornerRadius = 0
    field.layer.borderWidth = 0
    field.font = Theme.Typography.bodyEditorialMedium
    field.textColor = Theme.text
    field.textInsets = .zero
  }

  private func makeSectionLabel(title: String) -> UILabel {
    let label = UILabel()
    label.font = Theme.Typography.fieldLabel
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText.withAlphaComponent(0.82)
    label.text = title
    label.numberOfLines = 0
    return label
  }

  private func makeStackedSection(
    title: String,
    contentView: UIView,
    footerView: UIView? = nil
  ) -> UIView {
    contentView.translatesAutoresizingMaskIntoConstraints = false

    var arrangedSubviews: [UIView] = [makeSectionLabel(title: title), contentView]
    if let footerView {
      arrangedSubviews.append(footerView)
    }

    let stack = UIStackView(arrangedSubviews: arrangedSubviews)
    stack.axis = .vertical
    stack.spacing = 14
    stack.translatesAutoresizingMaskIntoConstraints = false

    let container = UIView()
    let separator = makeSeparator()
    container.addSubview(stack)
    container.addSubview(separator)

    NSLayoutConstraint.activate([
      stack.topAnchor.constraint(equalTo: container.topAnchor, constant: sectionTopInset),
      stack.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      stack.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),

      separator.topAnchor.constraint(equalTo: stack.bottomAnchor, constant: sectionBottomInset),
      separator.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      separator.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),
      separator.bottomAnchor.constraint(equalTo: container.bottomAnchor)
    ])

    return container
  }

  private func makeAboutSection() -> UIView {
    let footerSpacer = UIView()
    footerSpacer.setContentHuggingPriority(.defaultLow, for: .horizontal)

    let footerRow = UIStackView(arrangedSubviews: [footerSpacer, aboutCountLabel])
    footerRow.axis = .horizontal
    footerRow.alignment = .center
    footerRow.spacing = 12
    footerRow.translatesAutoresizingMaskIntoConstraints = false

    return makeStackedSection(
      title: Copy.bio,
      contentView: aboutField,
      footerView: footerRow
    )
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

  private func apply(user: User?) {
    currentUser = user

    let displayName = user?.displayName ?? L10n.tr("auth.brand")
    let username = user?.username ?? L10n.tr("profile.default_user_handle")
    let email = user?.email ?? ""
    let about = user?.about ?? ""

    nameLabel.text = displayName
    usernameValueLabel.text = "@\(username)"

    avatarView.configure(
      username: username,
      firstName: user?.firstName,
      lastName: user?.lastName,
      avatarURL: user?.avatarURL
    )

    displayNameField.text = displayName
    aboutField.text = about
    updateAboutCountLabel()

    originalProfile = EditableProfile(
      displayName: displayNameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "",
      email: email.trimmingCharacters(in: .whitespacesAndNewlines),
      about: aboutField.text.trimmingCharacters(in: .whitespacesAndNewlines)
    )
    updateButtonState()
  }

  private func refreshCurrentUser() async {
    do {
      let user = try await environment.api.fetchCurrentUser()
      environment.sessionStore.updateUser(user)
      await MainActor.run {
        apply(user: user)
      }
    } catch {
    }
  }

  private func currentProfileDraft() -> EditableProfile {
    EditableProfile(
      displayName: displayNameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "",
      email: currentUser?.email?.trimmingCharacters(in: .whitespacesAndNewlines) ?? originalProfile?.email ?? "",
      about: aboutField.text.trimmingCharacters(in: .whitespacesAndNewlines)
    )
  }

  private func splitDisplayName(_ displayName: String) -> (firstName: String, lastName: String)? {
    let parts = displayName
      .split(whereSeparator: \.isWhitespace)
      .map(String.init)

    guard let firstName = parts.first else { return nil }
    if parts.count > 1 {
      return (firstName, parts.dropFirst().joined(separator: " "))
    }

    if let currentLastName = currentUser?.lastName?.trimmingCharacters(in: .whitespacesAndNewlines),
       !currentLastName.isEmpty {
      return (firstName, currentLastName)
    }

    return nil
  }

  private func updateAboutCountLabel() {
    aboutCountLabel.text = L10n.tr("common.count_of_total", aboutField.text.count, maxAboutLength)
  }

  private func updateButtonState() {
    let profile = currentProfileDraft()
    let hasChanges = profile != originalProfile
    let validProfile = splitDisplayName(profile.displayName) != nil
      && AuthInputValidator.isValidEmail(profile.email)
      && profile.about.count <= maxAboutLength
    saveBarButtonItem.isEnabled = !isSavingProfile && hasChanges && validProfile
  }

  private func setSaving(_ saving: Bool) {
    isSavingProfile = saving
    displayNameField.isEnabled = !saving
    aboutField.isEditable = !saving
    navigationItem.rightBarButtonItem = saving ? activityIndicatorBarButtonItem : saveBarButtonItem
    updateButtonState()
  }

  @objc private func profileFieldChanged() {
    let trimmedDisplayName = displayNameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    nameLabel.text = trimmedDisplayName.isEmpty ? (currentUser?.displayName ?? L10n.tr("auth.brand")) : trimmedDisplayName
    updateButtonState()
  }

  @objc private func saveProfileTapped() {
    let profile = currentProfileDraft()

    guard let splitName = splitDisplayName(profile.displayName) else {
      presentError(Copy.invalidDisplayName)
      return
    }
    guard !profile.email.isEmpty else {
      presentError(Copy.missingProfileFields)
      return
    }
    guard AuthInputValidator.isValidEmail(profile.email) else {
      presentError(L10n.tr("error.invalid_email"))
      return
    }
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    setSaving(true)

    Task {
      do {
        let updatedUser = try await environment.api.updateCurrentUser(
          firstName: splitName.firstName,
          lastName: splitName.lastName,
          email: profile.email,
          about: profile.about.isEmpty ? nil : String(profile.about.prefix(maxAboutLength))
        )
        environment.sessionStore.updateUser(updatedUser)
        await MainActor.run {
          apply(user: updatedUser)
        }
      } catch {
        await MainActor.run {
          Haptics.shared.error()
          environment.feedbackCenter.showRecoverableActionFailure(.saveProfile)
        }
      }

      await MainActor.run {
        setSaving(false)
      }
    }
  }
}

extension ProfileSettingsViewController: UITextViewDelegate {
  func textViewDidChange(_ textView: UITextView) {
    if textView.text.count > maxAboutLength {
      textView.text = String(textView.text.prefix(maxAboutLength))
    }
    updateAboutCountLabel()
    updateButtonState()
  }

  func textView(
    _ textView: UITextView,
    shouldChangeTextIn range: NSRange,
    replacementText text: String
  ) -> Bool {
    guard let current = textView.text,
          let textRange = Range(range, in: current) else {
      return false
    }
    let updated = current.replacingCharacters(in: textRange, with: text)
    return updated.count <= maxAboutLength
  }
}
