import UIKit

final class SecuritySettingsViewController: BaseViewController {
  private let horizontalContentInset: CGFloat = 16
  private let fieldSectionTopInset: CGFloat = 10
  private let fieldSectionBottomInset: CGFloat = 14

  private enum Copy {
    static let title = L10n.tr("security.title")
    static let save = L10n.tr("security.update")
    static let currentPassword = L10n.tr("security.current_password")
    static let newPassword = L10n.tr("security.new_password")
    static let confirmNewPassword = L10n.tr("security.confirm_new_password")
    static let currentPasswordRequired = L10n.tr("security.current_password_required")
    static let passwordMismatch = L10n.tr("security.password_mismatch")
    static let successTitle = L10n.tr("security.updated_title")
    static let passwordUpdated = L10n.tr("security.updated_message")
  }

  private let environment: AppEnvironment
  private var isChangingPassword = false

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

  private let currentPasswordField = NFTextField(placeholder: Copy.currentPassword, secure: true)
  private let newPasswordField = NFTextField(placeholder: Copy.newPassword, secure: true)
  private let confirmPasswordField = NFTextField(placeholder: Copy.confirmNewPassword, secure: true)

  private lazy var logoutButton: UIButton = {
    let button = UIButton(type: .custom)
    button.translatesAutoresizingMaskIntoConstraints = false
    button.setTitle(L10n.tr("profile.logout"), for: .normal)
    button.setTitleColor(Theme.red, for: .normal)
    button.setTitleColor(Theme.red.withAlphaComponent(0.45), for: .disabled)
    button.titleLabel?.font = Theme.Typography.navigationAction
    button.titleLabel?.adjustsFontForContentSizeCategory = true
    button.contentHorizontalAlignment = .center
    button.heightAnchor.constraint(equalToConstant: 52).isActive = true
    button.addTarget(self, action: #selector(logoutTapped), for: .touchUpInside)
    return button
  }()

  private lazy var saveBarButtonItem: UIBarButtonItem = {
    makeNavigationTextBarButtonItem(
      title: Copy.save,
      accessibilityLabel: Copy.save,
      action: #selector(changePasswordTapped)
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
    updateButtonState()
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
    [currentPasswordField, newPasswordField, confirmPasswordField].forEach {
      configureFieldAppearance($0)
      $0.heightAnchor.constraint(equalToConstant: 40).isActive = true
      $0.addTarget(self, action: #selector(passwordFieldChanged), for: .editingChanged)
    }
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

    contentStack.addArrangedSubview(makeFormSection())
    contentStack.addArrangedSubview(makeSpacer(height: 32))
    contentStack.addArrangedSubview(makeSessionSection())
  }

  private func makeFormSection() -> UIView {
    let content = UIStackView(arrangedSubviews: [
      makeFlatFieldSection(title: Copy.currentPassword, inputView: currentPasswordField),
      makeFlatFieldSection(title: Copy.newPassword, inputView: newPasswordField),
      makeFlatFieldSection(title: Copy.confirmNewPassword, inputView: confirmPasswordField)
    ])
    content.axis = .vertical
    content.spacing = 0
    content.translatesAutoresizingMaskIntoConstraints = false

    let container = UIView()
    container.addSubview(content)

    NSLayoutConstraint.activate([
      content.topAnchor.constraint(equalTo: container.topAnchor),
      content.bottomAnchor.constraint(equalTo: container.bottomAnchor),
      content.leadingAnchor.constraint(equalTo: container.leadingAnchor),
      content.trailingAnchor.constraint(equalTo: container.trailingAnchor)
    ])

    return container
  }

  private func configureFieldAppearance(_ field: NFTextField) {
    field.backgroundColor = .clear
    field.layer.cornerRadius = 0
    field.layer.borderWidth = 0
    field.font = Theme.Typography.fieldValue
    field.textInsets = UIEdgeInsets(top: 12, left: 0, bottom: 12, right: 0)
  }

  private func makeMetadataLabel(title: String) -> UILabel {
    let label = UILabel()
    label.font = Theme.Typography.fieldLabel
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText.withAlphaComponent(0.82)
    label.text = title
    return label
  }

  private func makeFlatFieldSection(title: String, inputView: UIView) -> UIView {
    let stack = UIStackView(arrangedSubviews: [makeMetadataLabel(title: title), inputView])
    stack.axis = .vertical
    stack.spacing = 4
    stack.translatesAutoresizingMaskIntoConstraints = false

    let container = UIView()
    let separator = makeSeparator()
    container.addSubview(stack)
    container.addSubview(separator)

    NSLayoutConstraint.activate([
      stack.topAnchor.constraint(equalTo: container.topAnchor, constant: fieldSectionTopInset),
      stack.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      stack.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),

      separator.topAnchor.constraint(equalTo: stack.bottomAnchor, constant: fieldSectionBottomInset),
      separator.leadingAnchor.constraint(equalTo: container.leadingAnchor),
      separator.trailingAnchor.constraint(equalTo: container.trailingAnchor),
      separator.bottomAnchor.constraint(equalTo: container.bottomAnchor)
    ])

    return container
  }

  private func makeSessionSection() -> UIView {
    let stack = UIStackView(arrangedSubviews: [
      makeSeparator(),
      logoutButton,
      makeSeparator()
    ])
    stack.axis = .vertical
    stack.spacing = 0
    return stack
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

  private func updateButtonState() {
    let currentPassword = currentPasswordField.text ?? ""
    let newPassword = newPasswordField.text ?? ""
    let confirmPassword = confirmPasswordField.text ?? ""
    let canChange = !currentPassword.isEmpty && !newPassword.isEmpty && newPassword == confirmPassword
    saveBarButtonItem.isEnabled = !isChangingPassword && canChange
  }

  private func setChangingPassword(_ changing: Bool) {
    isChangingPassword = changing
    currentPasswordField.isEnabled = !changing
    newPasswordField.isEnabled = !changing
    confirmPasswordField.isEnabled = !changing
    navigationItem.rightBarButtonItem = changing ? activityIndicatorBarButtonItem : saveBarButtonItem
    updateButtonState()
  }

  @objc private func passwordFieldChanged() {
    updateButtonState()
  }

  @objc private func changePasswordTapped() {
    let currentPassword = currentPasswordField.text ?? ""
    let newPassword = newPasswordField.text ?? ""
    let confirmPassword = confirmPasswordField.text ?? ""

    guard !currentPassword.isEmpty, !newPassword.isEmpty else {
      presentError(Copy.currentPasswordRequired)
      return
    }
    guard newPassword == confirmPassword else {
      presentError(Copy.passwordMismatch)
      return
    }
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    setChangingPassword(true)

    Task {
      do {
        try await environment.api.changeCurrentUserPassword(
          currentPassword: currentPassword,
          newPassword: newPassword
        )
        await MainActor.run {
          currentPasswordField.text = nil
          newPasswordField.text = nil
          confirmPasswordField.text = nil
          updateButtonState()
          presentInfo(title: Copy.successTitle, message: Copy.passwordUpdated)
        }
      } catch {
        await MainActor.run {
          Haptics.shared.error()
          environment.feedbackCenter.showRecoverableActionFailure(.changePassword)
        }
      }

      await MainActor.run {
        setChangingPassword(false)
      }
    }
  }

  @objc private func logoutTapped() {
    logoutButton.isEnabled = false
    Task {
      do {
        try await environment.api.signOutCurrentSession()
      } catch {
        // Always clear local session even if server-side signout fails.
      }

      await MainActor.run {
        if let nav = navigationController, nav.presentingViewController != nil {
          nav.dismiss(animated: true) { [weak self] in
            self?.environment.sessionStore.clear()
          }
        } else {
          environment.sessionStore.clear()
        }
      }
    }
  }

  private func presentInfo(title: String, message: String) {
    let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
    alert.addAction(UIAlertAction(title: L10n.tr("common.ok"), style: .default))
    present(alert, animated: true)
  }
}
