import UIKit

@MainActor
final class AuthLandingViewController: BaseViewController {
  private let environment: AppEnvironment
  private var subtitleTimer: Timer?
  private var subtitleIndex = 0
  private lazy var subtitleMessages: [String] = [
    L10n.tr("auth.subtitle.1"),
    L10n.tr("auth.subtitle.2"),
    L10n.tr("auth.subtitle.3")
  ]

  private let bannerImageView: UIImageView = {
    let imageView = UIImageView()
    imageView.image = UIImage(named: "FeedBanner")
    imageView.contentMode = .scaleAspectFit
    imageView.clipsToBounds = true
    imageView.translatesAutoresizingMaskIntoConstraints = false
    return imageView
  }()

  private let subtitleLabel: UILabel = {
    let label = UILabel()
    label.text = L10n.tr("auth.subtitle.1")
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .center
    label.numberOfLines = 0
    return label
  }()

  private lazy var loginButton: NFButton = {
    let button = NFButton(title: L10n.tr("auth.login"))
    button.addTarget(self, action: #selector(openLogin), for: .touchUpInside)
    return button
  }()

  private lazy var signupButton: NFButton = {
    let button = NFButton(title: L10n.tr("auth.create_account"), filled: false)
    button.addTarget(self, action: #selector(openSignup), for: .touchUpInside)
    return button
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
//    title = L10n.tr("auth.welcome")

    let stack = UIStackView(arrangedSubviews: [bannerImageView, subtitleLabel, loginButton, signupButton])
    stack.axis = .vertical
    stack.spacing = 20
    stack.translatesAutoresizingMaskIntoConstraints = false

    view.addSubview(stack)

    NSLayoutConstraint.activate([
      stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
      stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
      stack.centerYAnchor.constraint(equalTo: view.centerYAnchor),
      bannerImageView.heightAnchor.constraint(equalToConstant: 120)
    ])
  }

  override func viewDidAppear(_ animated: Bool) {
    super.viewDidAppear(animated)
    startSubtitleRotation()
  }

  override func viewWillDisappear(_ animated: Bool) {
    super.viewWillDisappear(animated)
    stopSubtitleRotation()
  }

  @objc private func openLogin() {
    navigationController?.pushViewController(LoginViewController(environment: environment), animated: true)
  }

  @objc private func openSignup() {
    navigationController?.pushViewController(SignupViewController(environment: environment), animated: true)
  }

  private func startSubtitleRotation() {
    stopSubtitleRotation()
    subtitleLabel.text = subtitleMessages[subtitleIndex]
    let timer = Timer(timeInterval: 6, repeats: true) { [weak self] _ in
      self?.animateNextSubtitle()
    }
    timer.fireDate = Date().addingTimeInterval(5)
    RunLoop.main.add(timer, forMode: .common)
    subtitleTimer = timer
  }

  private func stopSubtitleRotation() {
    subtitleTimer?.invalidate()
    subtitleTimer = nil
  }

  private func animateNextSubtitle() {
    guard !subtitleMessages.isEmpty else { return }
    subtitleIndex = (subtitleIndex + 1) % subtitleMessages.count
    let nextText = subtitleMessages[subtitleIndex]

    UIView.transition(
      with: subtitleLabel,
      duration: 1.0,
      options: [.transitionCrossDissolve, .allowUserInteraction],
      animations: {
        self.subtitleLabel.text = nextText
      }
    )
  }
}
