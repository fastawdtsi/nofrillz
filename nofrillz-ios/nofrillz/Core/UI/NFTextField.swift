import UIKit

final class NFTextField: UITextField {
  var textInsets = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)

  init(placeholder: String, secure: Bool = false) {
    super.init(frame: .zero)
    self.placeholder = placeholder
    isSecureTextEntry = secure
    borderStyle = .none
    autocorrectionType = .no
    autocapitalizationType = .none
    clearButtonMode = .whileEditing
    textColor = Theme.text
    font = Theme.Typography.fieldValue
    adjustsFontForContentSizeCategory = true
    attributedPlaceholder = NSAttributedString(
      string: placeholder,
      attributes: Theme.Typography.placeholderAttributes()
    )
    Theme.applyTextInputStyle(to: self)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func textRect(forBounds bounds: CGRect) -> CGRect {
    bounds.inset(by: textInsets)
  }

  override func editingRect(forBounds bounds: CGRect) -> CGRect {
    bounds.inset(by: textInsets)
  }

  override func placeholderRect(forBounds bounds: CGRect) -> CGRect {
    bounds.inset(by: textInsets)
  }

  override func clearButtonRect(forBounds bounds: CGRect) -> CGRect {
    let defaultRect = super.clearButtonRect(forBounds: bounds)
    let horizontalOffset = max(0, textInsets.right - 4)
    return defaultRect.offsetBy(dx: -horizontalOffset, dy: 0)
  }
}
