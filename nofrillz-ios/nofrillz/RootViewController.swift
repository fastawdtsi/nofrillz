import UIKit

@MainActor
final class RootViewController: UIViewController {
  private let environment: AppEnvironment
  private var currentChild: UIViewController?
  private weak var currentConnectivityBannerView: ConnectivityBannerView?
  private var connectivityBannerDismissWorkItem: DispatchWorkItem?
  private var currentToastView: AppToastView?
  private var toastDismissWorkItem: DispatchWorkItem?
  private var currentTopSafeAreaInset: CGFloat = 0

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
    Theme.applyGlobalAppearance()
    view.backgroundColor = Theme.background
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleSessionChange),
      name: .sessionDidChange,
      object: nil
    )
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleConnectivityStatusChange(_:)),
      name: .connectivityStatusDidChange,
      object: nil
    )
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleToastMessageRequested(_:)),
      name: .toastMessageRequested,
      object: nil
    )
    renderCurrentFlow(animated: false)
    applyConnectivityStatus(
      environment.connectivityMonitor.status,
      previousStatus: .unknown,
      animated: false
    )
  }

  @objc private func handleSessionChange() {
    renderCurrentFlow(animated: true)
  }

  private func renderCurrentFlow(animated: Bool) {
    let nextVC: UIViewController

    if environment.sessionStore.isAuthenticated {
      nextVC = MainTabBarController(environment: environment)
    } else {
      let authLanding = AuthLandingViewController(environment: environment)
      nextVC = NoFrillzNavigationController(rootViewController: authLanding)
    }

    transition(to: nextVC, animated: animated)
  }

  private func transition(to next: UIViewController, animated: Bool) {
    if let currentChild {
      currentChild.willMove(toParent: nil)
      addChild(next)
      next.view.frame = view.bounds
      next.view.autoresizingMask = [.flexibleWidth, .flexibleHeight]

      transition(
        from: currentChild,
        to: next,
        duration: animated ? 0.2 : 0,
        options: [.transitionCrossDissolve, .curveEaseInOut]
      ) {
        currentChild.removeFromParent()
        next.didMove(toParent: self)
      }
    } else {
      addChild(next)
      next.view.frame = view.bounds
      next.view.autoresizingMask = [.flexibleWidth, .flexibleHeight]
      view.addSubview(next.view)
      next.didMove(toParent: self)
    }

    currentChild = next
    applyTopSafeAreaInset(currentTopSafeAreaInset, to: next)
  }

  @objc private func handleConnectivityStatusChange(_ notification: Notification) {
    guard let change = notification.connectivityStatusChange else { return }
    applyConnectivityStatus(change.currentStatus, previousStatus: change.previousStatus, animated: true)
  }

  @objc private func handleToastMessageRequested(_ notification: Notification) {
    guard let message = notification.toastMessage else { return }
    presentToast(message)
  }

  private func applyConnectivityStatus(
    _ status: ConnectivityStatus,
    previousStatus: ConnectivityStatus,
    animated: Bool
  ) {
    switch status {
    case .offline:
      presentConnectivityBanner(
        message: "You're offline. Some actions may be unavailable.",
        dismissAfter: nil,
        announce: true,
        animated: animated
      )
    case .online:
      if previousStatus == .offline {
        presentConnectivityBanner(
          message: "Back online.",
          dismissAfter: 1.8,
          announce: true,
          animated: animated
        )
      } else {
        dismissConnectivityBanner(animated: animated)
      }
    case .unknown:
      dismissConnectivityBanner(animated: animated)
    }
  }

  private func presentConnectivityBanner(
    message: String,
    dismissAfter: TimeInterval?,
    announce: Bool,
    animated: Bool
  ) {
    connectivityBannerDismissWorkItem?.cancel()

    if let banner = currentConnectivityBannerView {
      banner.removeFromSuperview()
      currentConnectivityBannerView = nil
    }

    let banner = ConnectivityBannerView(message: message)
    banner.alpha = animated ? 0 : 1
    banner.transform = animated && !UIAccessibility.isReduceMotionEnabled
      ? CGAffineTransform(translationX: 0, y: -8)
      : .identity
    view.addSubview(banner)

    NSLayoutConstraint.activate([
      banner.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 8),
      banner.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 12),
      banner.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -12)
    ])
    view.layoutIfNeeded()

    currentConnectivityBannerView = banner
    let bannerHeight = ceil(banner.systemLayoutSizeFitting(UIView.layoutFittingCompressedSize).height)
    currentTopSafeAreaInset = bannerHeight + 16
    applyTopSafeAreaInset(currentTopSafeAreaInset, to: currentChild)

    if animated {
      UIView.animate(withDuration: 0.18, delay: 0, options: [.curveEaseOut, .beginFromCurrentState]) {
        banner.alpha = 1
        banner.transform = .identity
      }
    }

    if announce {
      UIAccessibility.post(notification: .announcement, argument: message)
    }

    if let dismissAfter {
      let workItem = DispatchWorkItem { [weak self] in
        self?.dismissConnectivityBanner(animated: true)
      }
      connectivityBannerDismissWorkItem = workItem
      DispatchQueue.main.asyncAfter(deadline: .now() + dismissAfter, execute: workItem)
    }
  }

  private func dismissConnectivityBanner(animated: Bool) {
    connectivityBannerDismissWorkItem?.cancel()
    connectivityBannerDismissWorkItem = nil

    guard let banner = currentConnectivityBannerView else {
      currentTopSafeAreaInset = 0
      applyTopSafeAreaInset(0, to: currentChild)
      return
    }

    currentConnectivityBannerView = nil
    currentTopSafeAreaInset = 0
    applyTopSafeAreaInset(0, to: currentChild)

    let remove = {
      banner.removeFromSuperview()
    }

    guard animated else {
      remove()
      return
    }

    UIView.animate(withDuration: 0.16, delay: 0, options: [.curveEaseIn, .beginFromCurrentState]) {
      banner.alpha = 0
      if !UIAccessibility.isReduceMotionEnabled {
        banner.transform = CGAffineTransform(translationX: 0, y: -8)
      }
    } completion: { _ in
      remove()
    }
  }

  private func presentToast(_ message: ToastMessage) {
    toastDismissWorkItem?.cancel()

    if let toastView = currentToastView {
      toastView.removeFromSuperview()
      currentToastView = nil
    }

    let toastView = AppToastView(message: message.message)
    toastView.alpha = UIAccessibility.isReduceMotionEnabled ? 1 : 0
    if !UIAccessibility.isReduceMotionEnabled {
      toastView.transform = CGAffineTransform(translationX: 0, y: 8)
    }
    view.addSubview(toastView)

    let bottomInset = feedbackToastBottomInset
    NSLayoutConstraint.activate([
      toastView.centerXAnchor.constraint(equalTo: view.centerXAnchor),
      toastView.leadingAnchor.constraint(greaterThanOrEqualTo: view.leadingAnchor, constant: 20),
      toastView.trailingAnchor.constraint(lessThanOrEqualTo: view.trailingAnchor, constant: -20),
      toastView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -bottomInset)
    ])
    view.layoutIfNeeded()

    currentToastView = toastView

    if !UIAccessibility.isReduceMotionEnabled {
      UIView.animate(withDuration: 0.18, delay: 0, options: [.curveEaseOut, .beginFromCurrentState]) {
        toastView.alpha = 1
        toastView.transform = .identity
      }
    }

    if let accessibilityAnnouncement = message.accessibilityAnnouncement {
      UIAccessibility.post(notification: .announcement, argument: accessibilityAnnouncement)
    }

    let workItem = DispatchWorkItem { [weak self] in
      self?.dismissToast(animated: true)
    }
    toastDismissWorkItem = workItem
    DispatchQueue.main.asyncAfter(deadline: .now() + message.duration, execute: workItem)
  }

  private func dismissToast(animated: Bool) {
    toastDismissWorkItem?.cancel()
    toastDismissWorkItem = nil

    guard let toastView = currentToastView else { return }
    currentToastView = nil

    let remove = {
      toastView.removeFromSuperview()
    }

    guard animated, !UIAccessibility.isReduceMotionEnabled else {
      remove()
      return
    }

    UIView.animate(withDuration: 0.16, delay: 0, options: [.curveEaseIn, .beginFromCurrentState]) {
      toastView.alpha = 0
      toastView.transform = CGAffineTransform(translationX: 0, y: 8)
    } completion: { _ in
      remove()
    }
  }

  private func applyTopSafeAreaInset(_ inset: CGFloat, to child: UIViewController?) {
    child?.additionalSafeAreaInsets.top = inset
  }

  private var feedbackToastBottomInset: CGFloat {
    guard let mainTabBarController = currentChild as? MainTabBarController else {
      return view.safeAreaInsets.bottom + 16
    }
    return mainTabBarController.feedbackToastBottomInset
  }
}
