import UIKit

@MainActor
final class Haptics {
  static let shared = Haptics()

  private let softImpactGenerator = UIImpactFeedbackGenerator(style: .soft)
  private let lightImpactGenerator = UIImpactFeedbackGenerator(style: .light)
  private let successGenerator = UINotificationFeedbackGenerator()
  private let errorGenerator = UINotificationFeedbackGenerator()
  private let selectionGenerator = UISelectionFeedbackGenerator()

  private init() {
    softImpactGenerator.prepare()
    lightImpactGenerator.prepare()
    successGenerator.prepare()
    errorGenerator.prepare()
    selectionGenerator.prepare()
  }

  func softImpact() {
    softImpactGenerator.impactOccurred()
    softImpactGenerator.prepare()
  }

  func lightImpact() {
    lightImpactGenerator.impactOccurred()
    lightImpactGenerator.prepare()
  }

  func success() {
    successGenerator.notificationOccurred(.success)
    successGenerator.prepare()
  }

  func error() {
    errorGenerator.notificationOccurred(.error)
    errorGenerator.prepare()
  }

  func selection() {
    selectionGenerator.selectionChanged()
    selectionGenerator.prepare()
  }
}
