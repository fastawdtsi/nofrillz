import UIKit

final class PostTableViewCell: UITableViewCell {
  static let reuseID = "PostTableViewCell"
  private static let truncatedPreviewLineCount = 8

  var onLikeTap: (() -> Void)?
  var onShareTap: (() -> Void)?
  var onBookmarkTap: (() -> Void)?

  private let authorLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.editorialHeadline
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.primaryMetadataText
    label.isAccessibilityElement = false
    label.numberOfLines = 1
    label.lineBreakMode = .byTruncatingTail
    return label
  }()

  private let bodyPreviewLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.postBody
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.isAccessibilityElement = false
    label.numberOfLines = 0
    label.lineBreakMode = .byTruncatingTail
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()

  private let bodyContainerView = UIView()
  private let bodyFadeView = FeedPreviewFadeView()
  private var bodyFadeHeightConstraint: NSLayoutConstraint?
  private let continueReadingContainerView = UIView()
  private let continueReadingLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.postBody
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.alpha = 0.65
    label.isAccessibilityElement = false
    label.numberOfLines = 1
    label.lineBreakMode = .byTruncatingTail
    label.textAlignment = .center
    label.text = "Continue Reading"
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()
  private let continueReadingChevronView: UIImageView = {
    let imageView = UIImageView(image: AppIcon.chevronRight.image)
    imageView.tintColor = Theme.text
    imageView.alpha = 0.65
    imageView.contentMode = .scaleAspectFit
    imageView.isAccessibilityElement = false
    imageView.translatesAutoresizingMaskIntoConstraints = false
    return imageView
  }()

  private lazy var likeButton: UIButton = {
    let button = UIButton(type: .system)
    button.setImage(AppIcon.heart.image, for: .normal)
    button.tintColor = Theme.text
    button.imageView?.contentMode = .scaleAspectFit
    button.accessibilityLabel = "Respect"
    button.translatesAutoresizingMaskIntoConstraints = false
    button.addTarget(self, action: #selector(likeTapped), for: .touchUpInside)
    NSLayoutConstraint.activate([
      button.widthAnchor.constraint(equalToConstant: Theme.postActionIconSize),
      button.heightAnchor.constraint(equalToConstant: Theme.postActionIconSize)
    ])
    return button
  }()

  private lazy var shareButton: UIButton = {
    let button = UIButton(type: .system)
    button.setImage(AppIcon.share.image, for: .normal)
    button.tintColor = Theme.text
    button.imageView?.contentMode = .scaleAspectFit
    button.accessibilityLabel = L10n.tr("common.share")
    button.translatesAutoresizingMaskIntoConstraints = false
    button.addTarget(self, action: #selector(shareTapped), for: .touchUpInside)
    NSLayoutConstraint.activate([
      button.widthAnchor.constraint(equalToConstant: Theme.postActionIconSize),
      button.heightAnchor.constraint(equalToConstant: Theme.postActionIconSize)
    ])
    return button
  }()

  private lazy var bookmarkButton: UIButton = {
    let button = UIButton(type: .system)
    button.setImage(AppIcon.bookmark.image, for: .normal)
    button.tintColor = Theme.text
    button.imageView?.contentMode = .scaleAspectFit
    button.translatesAutoresizingMaskIntoConstraints = false
    button.addTarget(self, action: #selector(bookmarkTapped), for: .touchUpInside)
    NSLayoutConstraint.activate([
      button.widthAnchor.constraint(equalToConstant: Theme.postActionIconSize),
      button.heightAnchor.constraint(equalToConstant: Theme.postActionIconSize)
    ])
    return button
  }()

  private lazy var primaryActionsRow: UIStackView = {
    let row = UIStackView(arrangedSubviews: [likeButton, shareButton])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = Theme.postActionSpacing
    return row
  }()

  private let actionsSpacerView = UIView()

  private lazy var actionsRow: UIStackView = {
    actionsSpacerView.setContentHuggingPriority(.defaultLow, for: .horizontal)
    actionsSpacerView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

    let row = UIStackView(arrangedSubviews: [primaryActionsRow, actionsSpacerView, bookmarkButton])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = Theme.postActionSpacing
    return row
  }()

  private let separatorView: UIView = {
    let view = UIView()
    view.backgroundColor = Theme.separator
    view.translatesAutoresizingMaskIntoConstraints = false
    return view
  }()

  override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
    super.init(style: style, reuseIdentifier: reuseIdentifier)
    accessoryType = .none
    selectionStyle = .none
    accessibilityTraits.insert(.button)

    bodyContainerView.translatesAutoresizingMaskIntoConstraints = false
    bodyFadeView.translatesAutoresizingMaskIntoConstraints = false
    bodyFadeView.isHidden = true
    bodyContainerView.addSubview(bodyPreviewLabel)
    bodyContainerView.addSubview(bodyFadeView)

    NSLayoutConstraint.activate([
      bodyPreviewLabel.topAnchor.constraint(equalTo: bodyContainerView.topAnchor),
      bodyPreviewLabel.leadingAnchor.constraint(equalTo: bodyContainerView.leadingAnchor),
      bodyPreviewLabel.trailingAnchor.constraint(equalTo: bodyContainerView.trailingAnchor),
      bodyPreviewLabel.bottomAnchor.constraint(equalTo: bodyContainerView.bottomAnchor),

      bodyFadeView.leadingAnchor.constraint(equalTo: bodyContainerView.leadingAnchor),
      bodyFadeView.trailingAnchor.constraint(equalTo: bodyContainerView.trailingAnchor),
      bodyFadeView.bottomAnchor.constraint(equalTo: bodyContainerView.bottomAnchor)
    ])
    bodyFadeHeightConstraint = bodyFadeView.heightAnchor.constraint(equalToConstant: bodyFadeHeight)
    bodyFadeHeightConstraint?.isActive = true

    continueReadingContainerView.translatesAutoresizingMaskIntoConstraints = false
    continueReadingContainerView.isHidden = true
    continueReadingContainerView.addSubview(continueReadingLabel)
    continueReadingContainerView.addSubview(continueReadingChevronView)

    NSLayoutConstraint.activate([
      continueReadingChevronView.widthAnchor.constraint(equalToConstant: 24),
      continueReadingChevronView.heightAnchor.constraint(equalToConstant: 24),
      continueReadingLabel.topAnchor.constraint(equalTo: continueReadingContainerView.topAnchor),
      continueReadingLabel.bottomAnchor.constraint(equalTo: continueReadingContainerView.bottomAnchor),
      continueReadingLabel.leadingAnchor.constraint(greaterThanOrEqualTo: continueReadingContainerView.leadingAnchor),
      continueReadingChevronView.leadingAnchor.constraint(equalTo: continueReadingLabel.trailingAnchor, constant: 4),
      continueReadingChevronView.trailingAnchor.constraint(lessThanOrEqualTo: continueReadingContainerView.trailingAnchor),
      continueReadingChevronView.centerYAnchor.constraint(equalTo: continueReadingLabel.centerYAnchor, constant: 2),
      continueReadingLabel.centerXAnchor.constraint(equalTo: continueReadingContainerView.centerXAnchor, constant: -12)
    ])

    let stack = UIStackView(arrangedSubviews: [authorLabel, bodyContainerView, continueReadingContainerView])
    stack.axis = .vertical
    stack.spacing = 12
    stack.translatesAutoresizingMaskIntoConstraints = false

    contentView.addSubview(stack)
    actionsRow.translatesAutoresizingMaskIntoConstraints = false
    contentView.addSubview(actionsRow)
    contentView.addSubview(separatorView)

    NSLayoutConstraint.activate([
      stack.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 24),
      stack.bottomAnchor.constraint(equalTo: actionsRow.topAnchor, constant: -16),
      stack.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
      stack.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16),

      actionsRow.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
      actionsRow.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16),
      actionsRow.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -22),

      separatorView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor),
      separatorView.trailingAnchor.constraint(equalTo: contentView.trailingAnchor),
      separatorView.bottomAnchor.constraint(equalTo: contentView.bottomAnchor),
      separatorView.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func layoutSubviews() {
    super.layoutSubviews()
    bodyFadeHeightConstraint?.constant = bodyFadeHeight
  }

  override func prepareForReuse() {
    super.prepareForReuse()
    onLikeTap = nil
    onShareTap = nil
    onBookmarkTap = nil
    bodyPreviewLabel.attributedText = nil
    bodyFadeView.isHidden = true
    continueReadingContainerView.isHidden = true
    accessibilityLabel = nil
    accessibilityHint = nil
  }

  func configure(with post: Post, showAuthor: Bool = true, isBookmarkMutationInFlight: Bool = false) {
    authorLabel.isHidden = !showAuthor
    let preview = post.feedPreview
    authorLabel.attributedText = makeAuthorAttributedText(for: post, showsReadingTime: preview.isTruncated)
    bodyPreviewLabel.numberOfLines = preview.isTruncated ? Self.truncatedPreviewLineCount : 0
    bodyPreviewLabel.attributedText = makeBodyAttributedText(displayPreviewText(for: preview))
    bodyFadeView.isHidden = !preview.isTruncated
    continueReadingContainerView.isHidden = !preview.isTruncated

    configureLikeButton(isLiked: post.liked)
    let canShare = !(post.url?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
    shareButton.isEnabled = canShare
    shareButton.alpha = canShare ? 1 : 0.35
    configureBookmarkButton(for: post, isMutationInFlight: isBookmarkMutationInFlight)
    accessibilityLabel = makeAccessibilityLabel(for: post, preview: preview)
    accessibilityHint = makeAccessibilityHint(for: post, preview: preview)
  }

  @objc private func likeTapped() {
    onLikeTap?()
  }

  @objc private func shareTapped() {
    onShareTap?()
  }

  @objc private func bookmarkTapped() {
    onBookmarkTap?()
  }

  private func configureLikeButton(isLiked: Bool) {
    likeButton.setImage(isLiked ? AppIcon.heartFilled.image : AppIcon.heart.image, for: .normal)
    likeButton.tintColor = isLiked ? Theme.likedAccent : Theme.text
  }

  private func configureBookmarkButton(for post: Post, isMutationInFlight: Bool) {
    bookmarkButton.setImage(post.isBookmarked ? AppIcon.bookmarkFilled.image : AppIcon.bookmark.image, for: .normal)
    bookmarkButton.tintColor = post.isBookmarked ? Theme.bookmarkedAccent : Theme.text
    bookmarkButton.isEnabled = !isMutationInFlight
    bookmarkButton.alpha = isMutationInFlight ? 0.45 : 1
    bookmarkButton.accessibilityLabel = post.isBookmarked ? "Remove from Bookmarks" : "Save to Bookmarks"
    bookmarkButton.accessibilityValue = post.isBookmarked ? "Saved" : "Not saved"
  }

  private func makeBodyAttributedText(_ content: String) -> NSAttributedString {
    let paragraphStyle = NSMutableParagraphStyle()
    paragraphStyle.lineSpacing = Theme.postContentLineSpacing

    return NSAttributedString(
      string: content,
      attributes: [
        .font: Theme.Typography.postBody,
        .foregroundColor: Theme.text,
        .paragraphStyle: paragraphStyle
      ]
    )
  }

  private func makeAuthorAttributedText(for post: Post, showsReadingTime: Bool) -> NSAttributedString {
    let aiLabel = L10n.tr("common.ai_label")
    let name = post.authorDisplayName
    var metadataParts: [String] = []

    if let createdAt = post.createdAt {
      metadataParts.append(PostDateDisplay.shortString(from: createdAt))
    }

    if showsReadingTime, let readingTimeText = post.displayEstimatedReadingTimeText {
      metadataParts.append(readingTimeText)
    }

    if post.isAI {
      metadataParts.append(aiLabel)
    }

    guard !metadataParts.isEmpty else {
      return NSAttributedString(
        string: name,
        attributes: [
          .font: Theme.Typography.editorialHeadline,
          .foregroundColor: Theme.primaryMetadataText
        ]
      )
    }

    let metadata = metadataParts.joined(separator: " • ")
    let combined = "\(name) • \(metadata)"
    let attributed = NSMutableAttributedString(
      string: combined,
      attributes: [
        .font: Theme.Typography.editorialHeadline,
        .foregroundColor: Theme.primaryMetadataText
      ]
    )
    let metadataRange = NSRange(location: name.count, length: combined.count - name.count)
    attributed.addAttributes([
      .font: Theme.Typography.timestamp,
      .foregroundColor: Theme.secondaryText
    ], range: metadataRange)
    return attributed
  }

  private func displayPreviewText(for preview: PostPreview) -> String {
    guard preview.isTruncated, preview.text.hasSuffix("…") else {
      return preview.text
    }
    return String(preview.text.dropLast()).trimmingCharacters(in: .whitespacesAndNewlines)
  }

  private func makeAccessibilityLabel(for post: Post, preview: PostPreview) -> String {
    var components = [post.authorDisplayName]

    if let createdAt = post.createdAt {
      components.append(PostDateDisplay.shortString(from: createdAt))
    }

    if preview.isTruncated, let readingTimeText = post.displayEstimatedReadingTimeText {
      components.append(readingTimeText)
    }

    components.append(displayPreviewText(for: preview))
    return components
      .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
      .filter { !$0.isEmpty }
      .joined(separator: ". ")
  }

  private func makeAccessibilityHint(for post: Post, preview: PostPreview) -> String? {
    guard preview.isTruncated else { return nil }
    if let readingTimeText = post.displayEstimatedReadingTimeText {
      return "Double tap to read the full post. \(readingTimeText)."
    }
    return "Double tap to read the full post."
  }

  private var bodyFadeHeight: CGFloat {
    let lineHeight = bodyPreviewLabel.font.lineHeight
    return ceil((lineHeight * 3) + (Theme.postContentLineSpacing * 2) + 6)
  }

  func animateLikeTransition(duration: TimeInterval = 0.18) {
    likeButton.transform = .identity
    likeButton.alpha = 1
    UIView.animate(withDuration: duration / 2, delay: 0, options: [.curveEaseOut, .beginFromCurrentState]) {
      self.likeButton.transform = CGAffineTransform(scaleX: 1.04, y: 1.04)
      self.likeButton.alpha = 0.82
    } completion: { _ in
      UIView.animate(withDuration: duration / 2, delay: 0, options: [.curveEaseIn, .beginFromCurrentState]) {
        self.likeButton.transform = .identity
        self.likeButton.alpha = 1
      }
    }
  }
}

private final class FeedPreviewFadeView: UIView {
  override class var layerClass: AnyClass {
    CAGradientLayer.self
  }

  private var gradientLayer: CAGradientLayer {
    layer as! CAGradientLayer
  }

  override init(frame: CGRect) {
    super.init(frame: frame)
    isUserInteractionEnabled = false
    isAccessibilityElement = false
    backgroundColor = .clear
    configureGradient()
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func traitCollectionDidChange(_ previousTraitCollection: UITraitCollection?) {
    super.traitCollectionDidChange(previousTraitCollection)
    guard previousTraitCollection?.hasDifferentColorAppearance(comparedTo: traitCollection) ?? true else { return }
    configureGradient()
  }

  private func configureGradient() {
    let resolvedBackground = Theme.background.resolvedColor(with: traitCollection)
    gradientLayer.colors = [
      resolvedBackground.withAlphaComponent(0).cgColor,
      resolvedBackground.withAlphaComponent(0.18).cgColor,
      resolvedBackground.withAlphaComponent(0.72).cgColor,
      resolvedBackground.cgColor
    ]
    gradientLayer.locations = [0, 0.22, 0.62, 1]
    gradientLayer.startPoint = CGPoint(x: 0.5, y: 0)
    gradientLayer.endPoint = CGPoint(x: 0.5, y: 1)
  }
}
