import UIKit

final class NFButton: UIButton {
  init(title: String, filled: Bool = true) {
    super.init(frame: .zero)
    setTitle(title, for: .normal)
    setTitleColor(filled ? Theme.filledControlForeground : Theme.text, for: .normal)
    setTitleColor((filled ? Theme.filledControlForeground : Theme.text).withAlphaComponent(0.45), for: .disabled)
    titleLabel?.font = Theme.Typography.primaryButton
    titleLabel?.adjustsFontForContentSizeCategory = true
    if filled {
      Theme.applyFilledControlStyle(to: self, cornerRadius: 12)
    } else {
      Theme.applySurface(to: self, cornerRadius: 12, fillColor: Theme.elevatedSurfaceBackground)
    }
    heightAnchor.constraint(equalToConstant: 48).isActive = true
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }
}
