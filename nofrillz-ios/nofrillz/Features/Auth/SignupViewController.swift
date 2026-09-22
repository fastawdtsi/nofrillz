import UIKit

final class SignupViewController: BaseViewController {
  private let environment: AppEnvironment

  private let maxAboutLength = 100
  private let firstNameField = NFTextField(placeholder: L10n.tr("field.first_name"))
  private let lastNameField = NFTextField(placeholder: L10n.tr("field.last_name"))
  private let usernameField = NFTextField(placeholder: L10n.tr("field.username"))
  private let emailField = NFTextField(placeholder: L10n.tr("field.email"))
  private let passwordField = NFTextField(placeholder: L10n.tr("field.password"), secure: true)
  private let aboutTitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.fieldLabel
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.text = L10n.tr("field.about")
    return label
  }()

  private let aboutField: UITextView = {
    let textView = UITextView()
    textView.font = Theme.Typography.bodyEditorialSmall
    textView.adjustsFontForContentSizeCategory = true
    textView.textColor = Theme.text
    textView.textContainerInset = UIEdgeInsets(top: 10, left: 10, bottom: 10, right: 10)
    textView.translatesAutoresizingMaskIntoConstraints = false
    Theme.applyTextInputStyle(to: textView, cornerRadius: 12)
    return textView
  }()

  private let aboutCountLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.characterCounter
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .right
    label.text = L10n.tr("common.count_of_total", 0, 100)
    return label
  }()

  private lazy var signupButton: NFButton = {
    let button = NFButton(title: L10n.tr("auth.create_account"))
    button.addTarget(self, action: #selector(didTapSignup), for: .touchUpInside)
    return button
  }()

  private let activityIndicator: UIActivityIndicatorView = {
    let indicator = UIActivityIndicatorView(style: .medium)
    indicator.hidesWhenStopped = true
    indicator.color = Theme.text
    return indicator
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
//    title = L10n.tr("auth.sign_up")

    let aboutStack = UIStackView(arrangedSubviews: [aboutTitleLabel, aboutField, aboutCountLabel])
    aboutStack.axis = .vertical
    aboutStack.spacing = 6

    let stack = UIStackView(
      arrangedSubviews: [
        firstNameField,
        lastNameField,
        usernameField,
        emailField,
        passwordField,
        aboutStack,
        signupButton,
        activityIndicator
      ]
    )
    stack.axis = .vertical
    stack.spacing = 18
    stack.translatesAutoresizingMaskIntoConstraints = false

    firstNameField.autocapitalizationType = .words
    firstNameField.autocorrectionType = .no
    lastNameField.autocapitalizationType = .words
    lastNameField.autocorrectionType = .no
    aboutField.delegate = self
    aboutField.text = ""
    emailField.keyboardType = .emailAddress
    usernameField.keyboardType = .asciiCapable

    view.addSubview(stack)

    NSLayoutConstraint.activate([
      stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
      stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
      stack.centerYAnchor.constraint(equalTo: view.centerYAnchor),
      firstNameField.heightAnchor.constraint(equalToConstant: 44),
      lastNameField.heightAnchor.constraint(equalToConstant: 44),
      usernameField.heightAnchor.constraint(equalToConstant: 44),
      emailField.heightAnchor.constraint(equalToConstant: 44),
      passwordField.heightAnchor.constraint(equalToConstant: 44),
      aboutField.heightAnchor.constraint(equalToConstant: 88)
    ])
  }

  @objc private func didTapSignup() {
    guard let firstName = firstNameField.text?.trimmingCharacters(in: .whitespacesAndNewlines), !firstName.isEmpty,
          let lastName = lastNameField.text?.trimmingCharacters(in: .whitespacesAndNewlines), !lastName.isEmpty,
          let username = usernameField.text?.trimmingCharacters(in: .whitespacesAndNewlines), !username.isEmpty,
          let email = emailField.text?.trimmingCharacters(in: .whitespacesAndNewlines), !email.isEmpty,
          let password = passwordField.text, !password.isEmpty else {
      presentError(L10n.tr("error.enter_signup_fields"))
      return
    }
    guard AuthInputValidator.isValidUsername(username) else {
      presentError(L10n.tr("error.invalid_username"))
      return
    }
    guard AuthInputValidator.isValidEmail(email) else {
      presentError(L10n.tr("error.invalid_email"))
      return
    }
    let aboutText = aboutField.text.trimmingCharacters(in: .whitespacesAndNewlines)
    let about = aboutText.isEmpty ? nil : String(aboutText.prefix(maxAboutLength))
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    setLoading(true)

    Task {
      do {
        let auth = try await environment.api.signup(
          firstName: firstName,
          lastName: lastName,
          username: username,
          email: email,
          password: password,
          about: about
        )
        environment.sessionStore.saveSession(token: auth.token, refreshToken: auth.refreshToken, user: auth.user, userID: auth.userID)
      } catch {
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(.signUp)
      }

      setLoading(false)
    }
  }

  @MainActor
  private func setLoading(_ loading: Bool) {
    signupButton.isEnabled = !loading
    aboutField.isEditable = !loading
    if loading {
      activityIndicator.startAnimating()
    } else {
      activityIndicator.stopAnimating()
    }
  }
}

extension SignupViewController: UITextViewDelegate {
  func textViewDidChange(_ textView: UITextView) {
    let text = textView.text ?? ""
    if text.count > maxAboutLength {
      textView.text = String(text.prefix(maxAboutLength))
    }
    aboutCountLabel.text = L10n.tr("common.count_of_total", textView.text.count, maxAboutLength)
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
