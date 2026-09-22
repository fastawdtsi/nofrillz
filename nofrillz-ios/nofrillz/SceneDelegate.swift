import UIKit

@MainActor
class SceneDelegate: UIResponder, UIWindowSceneDelegate {
  var window: UIWindow?

  func scene(
    _ scene: UIScene,
    willConnectTo session: UISceneSession,
    options connectionOptions: UIScene.ConnectionOptions
  ) {
    guard let windowScene = scene as? UIWindowScene else { return }

    let environment = AppEnvironment.live()
    let rootViewController = RootViewController(environment: environment)

    let window = UIWindow(windowScene: windowScene)
    window.rootViewController = rootViewController
    window.makeKeyAndVisible()
    self.window = window

    PushNotificationManager.shared.handleLaunchNotificationResponse(connectionOptions.notificationResponse)
    PushNotificationManager.shared.configure(environment: environment, window: window)
  }

  func sceneDidBecomeActive(_ scene: UIScene) {
    PushNotificationManager.shared.attach(window: window)
  }
}
