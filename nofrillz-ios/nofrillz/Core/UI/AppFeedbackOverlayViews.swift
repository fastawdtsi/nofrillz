import UIKit

@MainActor
final class ConnectivityBannerView: UIView {
  private let label = UILabel()

  init(message: String) {
    super.init(frame: .zero)
    translatesAutoresizingMaskIntoConstraints = false

    let backgroundView = UIView()
    backgroundView.translatesAutoresizingMaskIntoConstraints = false
    Theme.applySurface(to: backgroundView, cornerRadius: 14, fillColor: Theme.elevatedSurfaceBackground)
    backgroundView.layer.borderWidth = 1 / UIScreen.main.scale
    backgroundView.layer.borderColor = Theme.surfaceBorder.cgColor
    addSubview(backgroundView)

    label.translatesAutoresizingMaskIntoConstraints = false
    label.font = Theme.Typography.metadata
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 0
    label.text = message
    backgroundView.addSubview(label)

    NSLayoutConstraint.activate([
      backgroundView.leadingAnchor.constraint(equalTo: leadingAnchor),
      backgroundView.trailingAnchor.constraint(equalTo: trailingAnchor),
      backgroundView.topAnchor.constraint(equalTo: topAnchor),
      backgroundView.bottomAnchor.constraint(equalTo: bottomAnchor),

      label.topAnchor.constraint(equalTo: backgroundView.topAnchor, constant: 10),
      label.bottomAnchor.constraint(equalTo: backgroundView.bottomAnchor, constant: -10),
      label.leadingAnchor.constraint(equalTo: backgroundView.leadingAnchor, constant: 14),
      label.trailingAnchor.constraint(equalTo: backgroundView.trailingAnchor, constant: -14)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }
}

@MainActor
final class AppToastView: UIView {
  private let label = UILabel()

  init(message: String) {
    super.init(frame: .zero)
    translatesAutoresizingMaskIntoConstraints = false

    let backgroundView = UIView()
    backgroundView.translatesAutoresizingMaskIntoConstraints = false
    Theme.applySurface(to: backgroundView, cornerRadius: 16, fillColor: Theme.elevatedSurfaceBackground)
    backgroundView.layer.borderWidth = 1 / UIScreen.main.scale
    backgroundView.layer.borderColor = Theme.surfaceBorder.cgColor
    addSubview(backgroundView)

    label.translatesAutoresizingMaskIntoConstraints = false
    label.font = Theme.Typography.caption
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 0
    label.textAlignment = .center
    label.text = message
    backgroundView.addSubview(label)

    NSLayoutConstraint.activate([
      backgroundView.leadingAnchor.constraint(equalTo: leadingAnchor),
      backgroundView.trailingAnchor.constraint(equalTo: trailingAnchor),
      backgroundView.topAnchor.constraint(equalTo: topAnchor),
      backgroundView.bottomAnchor.constraint(equalTo: bottomAnchor),

      label.topAnchor.constraint(equalTo: backgroundView.topAnchor, constant: 12),
      label.bottomAnchor.constraint(equalTo: backgroundView.bottomAnchor, constant: -12),
      label.leadingAnchor.constraint(equalTo: backgroundView.leadingAnchor, constant: 16),
      label.trailingAnchor.constraint(equalTo: backgroundView.trailingAnchor, constant: -16)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }
}
