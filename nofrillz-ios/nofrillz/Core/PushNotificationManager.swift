import UIKit
import UserNotifications

private enum PushDefaultsKey {
  static let deviceToken = "nofrillz.push.deviceToken"
  static let registeredUserID = "nofrillz.push.registeredUserID"
  static let registeredDeviceToken = "nofrillz.push.registeredDeviceToken"
}

private enum PushPreferenceKey {
  static let master = "nofrillz.notifications.master"
}

private struct PushNotificationPayload {
  let title: String?
  let body: String?
  let path: String?
  let routePath: String?

  init(title: String?, body: String?, path: String? = nil) {
    self.title = title
    self.body = body
    self.path = path
    self.routePath = path
  }

  init(userInfo: [AnyHashable: Any]) {
    if let aps = userInfo["aps"] as? [String: Any] {
      if let alert = aps["alert"] as? String {
        title = nil
        body = alert
      } else if let alert = aps["alert"] as? [String: Any] {
        title = alert["title"] as? String
        body = alert["body"] as? String
      } else {
        title = nil
        body = nil
      }
    } else {
      title = userInfo["title"] as? String
      body = userInfo["body"] as? String
    }

    path =
      (userInfo["path"] as? String)
      ?? (userInfo["url"] as? String)
      ?? (userInfo["deep_link"] as? String)
      ?? (userInfo["deepLink"] as? String)

    routePath = Self.resolveRoutePath(from: userInfo, explicitPath: path)
  }

  var hasVisibleContent: Bool {
    let trimmedTitle = title?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    let trimmedBody = body?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    return !trimmedTitle.isEmpty || !trimmedBody.isEmpty
  }

  private static func resolveRoutePath(from userInfo: [AnyHashable: Any], explicitPath: String?) -> String? {
    if let explicitPath, !explicitPath.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
      return explicitPath
    }

    let kind = firstString(
      in: userInfo,
      keys: ["type", "notification_type", "notificationType", "kind", "event"]
    )?.lowercased()

    if let postID = firstString(
      in: userInfo,
      keys: ["post_id", "postId", "target_post_id", "targetPostId", "resource_id", "resourceId"]
    ),
      !postID.isEmpty,
      shouldRouteToPost(for: kind)
    {
      return "/posts/\(postID)"
    }

    if let userID = firstString(
      in: userInfo,
      keys: ["user_id", "userId", "actor_id", "actorId", "follower_id", "followerId", "target_user_id", "targetUserId"]
    ),
      !userID.isEmpty,
      shouldRouteToUser(for: kind)
    {
      return "/users/\(userID)"
    }

    return nil
  }

  private static func shouldRouteToPost(for kind: String?) -> Bool {
    guard let kind else { return true }
    return kind.contains("like")
      || kind.contains("comment")
      || kind.contains("reply")
      || kind.contains("post")
  }

  private static func shouldRouteToUser(for kind: String?) -> Bool {
    guard let kind else { return true }
    return kind.contains("follow")
      || kind.contains("follower")
      || kind.contains("user")
  }

  private static func firstString(in userInfo: [AnyHashable: Any], keys: [String]) -> String? {
    for key in keys {
      if let value = userInfo[key] as? String {
        return value
      }
      if let number = userInfo[key] as? NSNumber {
        return number.stringValue
      }
    }
    return nil
  }
}

@MainActor
final class PushNotificationManager: NSObject {
  static let shared = PushNotificationManager()

  private let defaults = UserDefaults.standard
  private var environment: AppEnvironment?
  private weak var window: UIWindow?
  private var isBootstrapped = false
  private var isRegisteringDevice = false
  private var pendingTapPayload: PushNotificationPayload?
  private let bannerPresenter = InAppNotificationBannerPresenter()

  private override init() {
    super.init()
  }

  func bootstrap() {
    guard !isBootstrapped else { return }
    isBootstrapped = true
    UNUserNotificationCenter.current().delegate = self
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleSessionDidChange),
      name: .sessionDidChange,
      object: nil
    )
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleApplicationDidBecomeActive),
      name: UIApplication.didBecomeActiveNotification,
      object: nil
    )
  }

  func configure(environment: AppEnvironment, window: UIWindow?) {
    self.environment = environment
    self.window = window
    synchronizeRegistrationIfNeeded()
    processPendingTapPayloadIfPossible()
  }

  func attach(window: UIWindow?) {
    self.window = window
    processPendingTapPayloadIfPossible()
  }

  func didRegisterForRemoteNotifications(deviceToken: Data) {
    let token = deviceToken.map { String(format: "%02x", $0) }.joined()
    defaults.set(token, forKey: PushDefaultsKey.deviceToken)
    Task {
      await registerDeviceTokenIfNeeded(token)
    }
  }

  func didFailToRegisterForRemoteNotifications(error: Error) {
#if DEBUG
    print("[NoFrillzDebug][Push] Failed to register for remote notifications: \(error)")
#endif
  }

  func handleNotificationResponse(_ response: UNNotificationResponse) {
    pendingTapPayload = PushNotificationPayload(userInfo: response.notification.request.content.userInfo)
    processPendingTapPayloadIfPossible()
  }

  func handleLaunchNotificationResponse(_ response: UNNotificationResponse?) {
    guard let response else { return }
    pendingTapPayload = PushNotificationPayload(userInfo: response.notification.request.content.userInfo)
  }

  func showPreviewBanner() {
    guard let window = activeWindow() else { return }
    let payload = PushNotificationPayload(
      title: L10n.tr("notifications.preview.banner_title"),
      body: L10n.tr("notifications.preview.banner_body")
    )
    bannerPresenter.show(payload: payload, in: window, onTap: nil)
  }

  private var cachedDeviceToken: String? {
    defaults.string(forKey: PushDefaultsKey.deviceToken)
  }

  private var registeredUserID: String? {
    defaults.string(forKey: PushDefaultsKey.registeredUserID)
  }

  private var registeredDeviceToken: String? {
    defaults.string(forKey: PushDefaultsKey.registeredDeviceToken)
  }

  private var localPushPreferenceEnabled: Bool {
    defaults.object(forKey: PushPreferenceKey.master) as? Bool ?? true
  }

  @objc private func handleSessionDidChange() {
    synchronizeRegistrationIfNeeded()
    processPendingTapPayloadIfPossible()
  }

  @objc private func handleApplicationDidBecomeActive() {
    synchronizeRegistrationIfNeeded()
    processPendingTapPayloadIfPossible()
  }

  private func synchronizeRegistrationIfNeeded() {
    guard let environment, environment.sessionStore.isAuthenticated else { return }
    Task {
      await requestAuthorizationAndRegisterIfNeeded()
    }
  }

  private func requestAuthorizationAndRegisterIfNeeded() async {
    guard let environment, let currentUserID = environment.sessionStore.currentUser?.id, !currentUserID.isEmpty else {
      return
    }

    if registeredUserID == currentUserID, registeredDeviceToken == cachedDeviceToken, cachedDeviceToken != nil {
      return
    }

    let settings = await UNUserNotificationCenter.current().notificationSettings()
    switch settings.authorizationStatus {
    case .notDetermined:
      do {
        let granted = try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge])
        guard granted else { return }
        UIApplication.shared.registerForRemoteNotifications()
      } catch {
#if DEBUG
        print("[NoFrillzDebug][Push] Authorization request failed: \(error)")
#endif
      }
    case .authorized, .provisional, .ephemeral:
      UIApplication.shared.registerForRemoteNotifications()
      if let token = cachedDeviceToken {
        await registerDeviceTokenIfNeeded(token)
      }
    case .denied:
      return
    @unknown default:
      return
    }
  }

  private func registerDeviceTokenIfNeeded(_ token: String) async {
    guard let environment, let currentUserID = environment.sessionStore.currentUser?.id, !currentUserID.isEmpty else {
      return
    }
    guard !token.isEmpty else { return }
    guard !(registeredUserID == currentUserID && registeredDeviceToken == token) else { return }
    guard !isRegisteringDevice else { return }

    isRegisteringDevice = true
    defer { isRegisteringDevice = false }

    do {
      try await environment.api.registerPushDevice(token: token)
      defaults.set(currentUserID, forKey: PushDefaultsKey.registeredUserID)
      defaults.set(token, forKey: PushDefaultsKey.registeredDeviceToken)
    } catch {
#if DEBUG
      print("[NoFrillzDebug][Push] Device registration failed: \(error)")
#endif
    }
  }

  private func processPendingTapPayloadIfPossible() {
    guard let payload = pendingTapPayload else { return }
    guard environment?.sessionStore.isAuthenticated == true else { return }

    if let path = payload.routePath?.trimmingCharacters(in: .whitespacesAndNewlines), !path.isEmpty, route(to: path) {
      pendingTapPayload = nil
      return
    }

    guard payload.hasVisibleContent, let window = activeWindow() else { return }
    pendingTapPayload = nil
    bannerPresenter.show(payload: payload, in: window, onTap: nil)
  }

  private func route(to path: String) -> Bool {
    guard let mainTabBarController = findMainTabBarController(in: activeWindow()?.rootViewController) else {
      return false
    }
    return mainTabBarController.openNotificationPath(path)
  }

  private func activeWindow() -> UIWindow? {
    if let window {
      return window
    }

    let connectedScenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
    for scene in connectedScenes {
      if let keyWindow = scene.windows.first(where: \.isKeyWindow) {
        return keyWindow
      }
    }
    return nil
  }

  private func findMainTabBarController(in root: UIViewController?) -> MainTabBarController? {
    guard let root else { return nil }
    if let controller = root as? MainTabBarController {
      return controller
    }
    for child in root.children {
      if let found = findMainTabBarController(in: child) {
        return found
      }
    }
    if let navigationController = root as? UINavigationController {
      return findMainTabBarController(in: navigationController.visibleViewController)
    }
    if let presented = root.presentedViewController {
      return findMainTabBarController(in: presented)
    }
    return nil
  }
}

extension PushNotificationManager: UNUserNotificationCenterDelegate {
  nonisolated func userNotificationCenter(
    _ center: UNUserNotificationCenter,
    willPresent notification: UNNotification,
    withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
  ) {
    Task { @MainActor in
      let payload = PushNotificationPayload(userInfo: notification.request.content.userInfo)
      if localPushPreferenceEnabled, let window = activeWindow(), payload.hasVisibleContent {
        bannerPresenter.show(payload: payload, in: window) { [weak self] in
          guard let self, let path = payload.routePath else { return }
          _ = self.route(to: path)
        }
      }
      completionHandler([])
    }
  }

  nonisolated func userNotificationCenter(
    _ center: UNUserNotificationCenter,
    didReceive response: UNNotificationResponse,
    withCompletionHandler completionHandler: @escaping () -> Void
  ) {
    Task { @MainActor in
      handleNotificationResponse(response)
      completionHandler()
    }
  }
}

@MainActor
private final class InAppNotificationBannerPresenter {
  private weak var currentBanner: InAppNotificationBannerView?
  private var dismissWorkItem: DispatchWorkItem?

  func show(payload: PushNotificationPayload, in window: UIWindow, onTap: (() -> Void)? = nil) {
    dismiss(animated: false)

    let banner = InAppNotificationBannerView(payload: payload)
    banner.translatesAutoresizingMaskIntoConstraints = false
    banner.alpha = 0
    banner.transform = CGAffineTransform(translationX: 0, y: -12)
    banner.onTap = { [weak self] in
      self?.dismiss(animated: true)
      onTap?()
    }
    banner.onSwipeDismiss = { [weak self] in
      self?.dismiss(animated: true)
    }

    window.addSubview(banner)
    NSLayoutConstraint.activate([
      banner.leadingAnchor.constraint(equalTo: window.leadingAnchor, constant: 12),
      banner.trailingAnchor.constraint(equalTo: window.trailingAnchor, constant: -12),
      banner.topAnchor.constraint(equalTo: window.safeAreaLayoutGuide.topAnchor, constant: 8)
    ])
    window.layoutIfNeeded()

    currentBanner = banner
    Haptics.shared.softImpact()
    UIView.animate(withDuration: 0.18, delay: 0, options: [.curveEaseOut, .beginFromCurrentState]) {
      banner.alpha = 1
      banner.transform = .identity
    }

    let workItem = DispatchWorkItem { [weak self] in
      self?.dismiss(animated: true)
    }
    dismissWorkItem = workItem
    DispatchQueue.main.asyncAfter(deadline: .now() + 4.5, execute: workItem)
  }

  func dismiss(animated: Bool) {
    dismissWorkItem?.cancel()
    dismissWorkItem = nil

    guard let banner = currentBanner else { return }
    currentBanner = nil

    let remove = {
      banner.removeFromSuperview()
    }

    if animated {
      UIView.animate(withDuration: 0.16, delay: 0, options: [.curveEaseIn, .beginFromCurrentState]) {
        banner.alpha = 0
        banner.transform = CGAffineTransform(translationX: 0, y: -8)
      } completion: { _ in
        remove()
      }
    } else {
      remove()
    }
  }
}

@MainActor
private final class InAppNotificationBannerView: UIControl {
  var onTap: (() -> Void)?
  var onSwipeDismiss: (() -> Void)?

  private let titleLabel = UILabel()
  private let bodyLabel = UILabel()
  private lazy var panGestureRecognizer: UIPanGestureRecognizer = {
    let gestureRecognizer = UIPanGestureRecognizer(target: self, action: #selector(handlePanGesture(_:)))
    gestureRecognizer.cancelsTouchesInView = true
    return gestureRecognizer
  }()

  init(payload: PushNotificationPayload) {
    super.init(frame: .zero)
    addTarget(self, action: #selector(didTapBanner), for: .touchUpInside)
    addGestureRecognizer(panGestureRecognizer)

    let backgroundView = UIView()
    backgroundView.translatesAutoresizingMaskIntoConstraints = false
    Theme.applySurface(to: backgroundView, cornerRadius: 18, fillColor: Theme.elevatedSurfaceBackground)
    addSubview(backgroundView)

    titleLabel.font = Theme.Typography.inlineTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.numberOfLines = 2
    titleLabel.text = payload.title?.trimmingCharacters(in: .whitespacesAndNewlines)
    titleLabel.isHidden = titleLabel.text?.isEmpty ?? true

    bodyLabel.font = Theme.Typography.metadata
    bodyLabel.adjustsFontForContentSizeCategory = true
    bodyLabel.textColor = Theme.secondaryText
    bodyLabel.numberOfLines = 3
    bodyLabel.text = payload.body?.trimmingCharacters(in: .whitespacesAndNewlines)
    if titleLabel.isHidden, let bodyText = bodyLabel.text, !bodyText.isEmpty {
      bodyLabel.font = Theme.Typography.caption
      bodyLabel.textColor = Theme.text
    }

    let labelsStack = UIStackView(arrangedSubviews: [titleLabel, bodyLabel])
    labelsStack.translatesAutoresizingMaskIntoConstraints = false
    labelsStack.axis = .vertical
    labelsStack.alignment = .fill
    labelsStack.spacing = 4
    backgroundView.addSubview(labelsStack)

    NSLayoutConstraint.activate([
      backgroundView.topAnchor.constraint(equalTo: topAnchor),
      backgroundView.bottomAnchor.constraint(equalTo: bottomAnchor),
      backgroundView.leadingAnchor.constraint(equalTo: leadingAnchor),
      backgroundView.trailingAnchor.constraint(equalTo: trailingAnchor),

      labelsStack.topAnchor.constraint(equalTo: backgroundView.topAnchor, constant: 14),
      labelsStack.bottomAnchor.constraint(equalTo: backgroundView.bottomAnchor, constant: -14),
      labelsStack.leadingAnchor.constraint(equalTo: backgroundView.leadingAnchor, constant: 16),
      labelsStack.trailingAnchor.constraint(equalTo: backgroundView.trailingAnchor, constant: -16)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  @objc private func didTapBanner() {
    onTap?()
  }

  @objc private func handlePanGesture(_ gestureRecognizer: UIPanGestureRecognizer) {
    let translation = gestureRecognizer.translation(in: self)
    let verticalTranslation = min(translation.y, 0)

    switch gestureRecognizer.state {
    case .changed:
      transform = CGAffineTransform(translationX: 0, y: verticalTranslation)
      let progress = min(abs(verticalTranslation) / 60, 1)
      alpha = 1 - (progress * 0.45)

    case .ended, .cancelled, .failed:
      let velocity = gestureRecognizer.velocity(in: self)
      let shouldDismiss = verticalTranslation < -28 || velocity.y < -280

      if shouldDismiss {
        onSwipeDismiss?()
      } else {
        UIView.animate(withDuration: 0.18, delay: 0, options: [.curveEaseOut, .beginFromCurrentState]) {
          self.transform = .identity
          self.alpha = 1
        }
      }

    default:
      break
    }
  }
}
