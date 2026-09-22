import UIKit

final class UserCell: UITableViewCell {
  static let reuseID = "UserCell"

  var onFollowTap: (() -> Void)?

  private let fullNameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.editorialHeadline
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.numberOfLines = 1
    label.lineBreakMode = .byTruncatingTail
    return label
  }()

  private let usernameLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.username
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 1
    label.lineBreakMode = .byTruncatingTail
    return label
  }()

  private let avatarView: AvatarView = {
    let view = AvatarView()
    view.widthAnchor.constraint(equalToConstant: 36).isActive = true
    view.heightAnchor.constraint(equalToConstant: 36).isActive = true
    return view
  }()

  private lazy var followButton: UIButton = {
    let button = UIButton(type: .custom)
    button.setImage(AppIcon.plus.image, for: .normal)
    button.tintColor = Theme.filledControlForeground
    button.imageView?.contentMode = .scaleAspectFit
    button.contentEdgeInsets = UIEdgeInsets(top: 6, left: 6, bottom: 6, right: 6)
    Theme.applyFilledControlStyle(to: button, cornerRadius: 14)
    button.widthAnchor.constraint(equalToConstant: 28).isActive = true
    button.heightAnchor.constraint(equalToConstant: 28).isActive = true
    button.addTarget(self, action: #selector(followTapped), for: .touchUpInside)
    return button
  }()

  override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
    super.init(style: style, reuseIdentifier: reuseIdentifier)
    selectionStyle = .default

    let labels = UIStackView(arrangedSubviews: [fullNameLabel, usernameLabel])
    labels.axis = .vertical
    labels.spacing = 2

    let spacer = UIView()
    spacer.translatesAutoresizingMaskIntoConstraints = false
    let row = UIStackView(arrangedSubviews: [avatarView, labels, spacer, followButton])
    row.axis = .horizontal
    row.alignment = .top
    row.spacing = 12
    row.translatesAutoresizingMaskIntoConstraints = false

    labels.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    followButton.setContentCompressionResistancePriority(.required, for: .horizontal)
    avatarView.setContentCompressionResistancePriority(.required, for: .horizontal)
    avatarView.setContentHuggingPriority(.required, for: .horizontal)

    contentView.addSubview(row)

    NSLayoutConstraint.activate([
      row.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 12),
      row.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -12),
      row.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
      row.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  func configure(user: User, isFollowing: Bool, followEnabled: Bool) {
    avatarView.configure(
      username: user.username,
      firstName: user.firstName,
      lastName: user.lastName,
      avatarURL: user.avatarURL
    )
    fullNameLabel.text = user.fullName ?? user.username
    usernameLabel.text = "@\(user.username)"
    if isFollowing {
      followButton.setTitle(nil, for: .normal)
      followButton.setImage(AppIcon.check.image, for: .normal)
      Theme.applySurface(to: followButton, cornerRadius: 14, fillColor: Theme.elevatedSurfaceBackground)
      followButton.tintColor = Theme.text
    } else {
      followButton.setTitle(nil, for: .normal)
      followButton.setImage(AppIcon.plus.image, for: .normal)
      Theme.applyFilledControlStyle(to: followButton, cornerRadius: 14)
      followButton.tintColor = Theme.filledControlForeground
    }
    followButton.isEnabled = followEnabled
    followButton.alpha = followEnabled ? 1 : 0.6
  }

  @objc private func followTapped() {
    onFollowTap?()
  }
}
