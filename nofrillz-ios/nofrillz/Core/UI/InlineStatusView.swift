import UIKit

@MainActor
final class InlineStatusView: UIView {
  private let titleLabel = UILabel()
  private let messageLabel = UILabel()
  private let actionButton = UIButton(type: .system)
  private let stackView = UIStackView()

  var onActionTap: (() -> Void)?

  override init(frame: CGRect) {
    super.init(frame: frame)
    translatesAutoresizingMaskIntoConstraints = false

    titleLabel.font = Theme.Typography.bodyEditorialLarge
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.textAlignment = .center
    titleLabel.numberOfLines = 0

    messageLabel.font = Theme.Typography.helperText
    messageLabel.adjustsFontForContentSizeCategory = true
    messageLabel.textColor = Theme.secondaryText
    messageLabel.textAlignment = .center
    messageLabel.numberOfLines = 0

    actionButton.titleLabel?.font = Theme.Typography.primaryAction
    actionButton.titleLabel?.adjustsFontForContentSizeCategory = true
    actionButton.setTitleColor(Theme.text, for: .normal)
    actionButton.setTitleColor(Theme.disabledActionText, for: .disabled)
    actionButton.addTarget(self, action: #selector(actionTapped), for: .touchUpInside)

    stackView.axis = .vertical
    stackView.alignment = .center
    stackView.spacing = 12
    stackView.translatesAutoresizingMaskIntoConstraints = false
    stackView.isLayoutMarginsRelativeArrangement = true
    stackView.layoutMargins = UIEdgeInsets(top: 32, left: 24, bottom: 32, right: 24)
    stackView.addArrangedSubview(titleLabel)
    stackView.addArrangedSubview(messageLabel)
    stackView.addArrangedSubview(actionButton)

    addSubview(stackView)
    NSLayoutConstraint.activate([
      stackView.centerYAnchor.constraint(equalTo: centerYAnchor),
      stackView.leadingAnchor.constraint(equalTo: leadingAnchor),
      stackView.trailingAnchor.constraint(equalTo: trailingAnchor)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  func configure(
    title: String,
    message: String,
    actionTitle: String? = nil
  ) {
    titleLabel.text = title
    messageLabel.text = message

    if let actionTitle, !actionTitle.isEmpty {
      actionButton.isHidden = false
      actionButton.setTitle(actionTitle, for: .normal)
    } else {
      actionButton.isHidden = true
      actionButton.setTitle(nil, for: .normal)
    }
  }

  @objc private func actionTapped() {
    onActionTap?()
  }
}
