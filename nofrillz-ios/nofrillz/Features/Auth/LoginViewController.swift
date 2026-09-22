import UIKit

final class LoginViewController: BaseViewController {
  private let environment: AppEnvironment

  private let emailField = NFTextField(placeholder: L10n.tr("field.email"))
  private let passwordField = NFTextField(placeholder: L10n.tr("field.password"), secure: true)
  private lazy var loginButton: NFButton = {
    let button = NFButton(title: L10n.tr("auth.login"))
    button.addTarget(self, action: #selector(didTapLogin), for: .touchUpInside)
    return button
  }()

  private let activityIndicator: UIActivityIndicatorView = {
    let indicator = UIActivityIndicatorView(style: .medium)
    indicator.hidesWhenStopped = true
    indicator.color = .black
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
//    title = L10n.tr("auth.login")

    let stack = UIStackView(arrangedSubviews: [emailField, passwordField, loginButton, activityIndicator])
    stack.axis = .vertical
    stack.spacing = 18
    stack.translatesAutoresizingMaskIntoConstraints = false
    emailField.keyboardType = .emailAddress

    view.addSubview(stack)
    NSLayoutConstraint.activate([
      stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
      stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
      stack.centerYAnchor.constraint(equalTo: view.centerYAnchor),
      emailField.heightAnchor.constraint(equalToConstant: 44),
      passwordField.heightAnchor.constraint(equalToConstant: 44)
    ])
  }

  @objc private func didTapLogin() {
    guard let email = emailField.text?.trimmingCharacters(in: .whitespacesAndNewlines), !email.isEmpty,
          let password = passwordField.text, !password.isEmpty else {
      presentError(L10n.tr("error.enter_email_password"))
      return
    }
    guard AuthInputValidator.isValidEmail(email) else {
      presentError(L10n.tr("error.invalid_email"))
      return
    }
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    setLoading(true)

    Task {
      do {
        let auth = try await environment.api.login(email: email, password: password)
        environment.sessionStore.saveSession(token: auth.token, refreshToken: auth.refreshToken, user: auth.user, userID: auth.userID)
      } catch {
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(.signIn)
      }
      setLoading(false)
    }
  }

  @MainActor
  private func setLoading(_ loading: Bool) {
    loginButton.isEnabled = !loading
    if loading {
      activityIndicator.startAnimating()
    } else {
      activityIndicator.stopAnimating()
    }
  }
}
