import UIKit

@MainActor
final class NoFrillzNavigationController: UINavigationController, UIGestureRecognizerDelegate {
  override func viewDidLoad() {
    super.viewDidLoad()
    interactivePopGestureRecognizer?.delegate = self
    interactivePopGestureRecognizer?.isEnabled = true
  }

  func gestureRecognizerShouldBegin(_ gestureRecognizer: UIGestureRecognizer) -> Bool {
    guard gestureRecognizer === interactivePopGestureRecognizer else { return true }
    guard viewControllers.count > 1 else { return false }

    if let isTransitioning = value(forKey: "_isTransitioning") as? Bool, isTransitioning {
      return false
    }

    return true
  }
}
