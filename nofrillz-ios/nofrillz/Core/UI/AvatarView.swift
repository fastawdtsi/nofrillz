import UIKit

private final class AvatarInitialLabel: UILabel {
  var textInsets = UIEdgeInsets(top: 4, left: 4, bottom: 4, right: 4)

  override func textRect(forBounds bounds: CGRect, limitedToNumberOfLines numberOfLines: Int) -> CGRect {
    let insetBounds = bounds.inset(by: textInsets)
    let textRect = super.textRect(forBounds: insetBounds, limitedToNumberOfLines: numberOfLines)
    return CGRect(
      x: textRect.origin.x - textInsets.left,
      y: textRect.origin.y - textInsets.top,
      width: textRect.width + textInsets.left + textInsets.right,
      height: textRect.height + textInsets.top + textInsets.bottom
    )
  }

  override func drawText(in rect: CGRect) {
    super.drawText(in: rect.inset(by: textInsets))
  }
}

@MainActor
final class AvatarView: UIView {
  private static let imageCache = NSCache<NSString, UIImage>()

  private let imageView: UIImageView = {
    let imageView = UIImageView()
    imageView.contentMode = .scaleAspectFill
    imageView.clipsToBounds = true
    imageView.translatesAutoresizingMaskIntoConstraints = false
    return imageView
  }()

  private let initialLabel: AvatarInitialLabel = {
    let label = AvatarInitialLabel()
    label.font = UIFont(name: "Allura-Regular", size: 16) ?? Theme.contentFont(ofSize: 16, weight: .regular)
    label.textColor = Theme.text
    label.textAlignment = .center
    label.numberOfLines = 1
    label.adjustsFontSizeToFitWidth = true
    label.minimumScaleFactor = 0.55
    label.baselineAdjustment = .alignCenters
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()

  private var imageTask: Task<Void, Never>?
  private var currentURLString: String?

  override init(frame: CGRect) {
    super.init(frame: frame)
    translatesAutoresizingMaskIntoConstraints = false
    clipsToBounds = true
    Theme.applySurface(to: self, cornerRadius: 0, fillColor: Theme.elevatedSurfaceBackground)

    addSubview(imageView)
    addSubview(initialLabel)

    NSLayoutConstraint.activate([
      imageView.topAnchor.constraint(equalTo: topAnchor),
      imageView.bottomAnchor.constraint(equalTo: bottomAnchor),
      imageView.leadingAnchor.constraint(equalTo: leadingAnchor),
      imageView.trailingAnchor.constraint(equalTo: trailingAnchor)
    ])
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  deinit {
    imageTask?.cancel()
  }

  override func layoutSubviews() {
    super.layoutSubviews()
    let diameter = min(bounds.width, bounds.height)
    layer.cornerRadius = diameter / 2
    if initialLabel.isHidden {
      Theme.applySurface(to: self, cornerRadius: diameter / 2, fillColor: .clear)
      layer.borderWidth = 0
    } else {
      Theme.applySurface(to: self, cornerRadius: diameter / 2, fillColor: Theme.elevatedSurfaceBackground)
    }

    let initialsCount = initialLabel.text?.count ?? 1
    let fontSize = initialsCount > 1 ? max(14, diameter * 0.5) : max(16, diameter * 0.78)
    initialLabel.font = UIFont(name: "Allura-Regular", size: fontSize)
      ?? Theme.contentFont(ofSize: fontSize, weight: .regular)
    let labelInset = max(4, diameter * 0.08)
    initialLabel.frame = bounds.insetBy(dx: labelInset, dy: labelInset)
  }

  func configure(username: String, firstName: String? = nil, lastName: String? = nil, avatarURL: String?) {
    imageTask?.cancel()
    currentURLString = nil
    initialLabel.text = AvatarView.initials(username: username, firstName: firstName, lastName: lastName)
    initialLabel.isHidden = false
    imageView.image = nil
    Theme.applySurface(to: self, cornerRadius: bounds.width / 2, fillColor: Theme.elevatedSurfaceBackground)

    guard let avatarURL,
          let normalizedURL = AvatarView.normalizedURL(from: avatarURL) else {
      return
    }

    let cacheKey = normalizedURL.absoluteString as NSString
    if let cachedImage = AvatarView.imageCache.object(forKey: cacheKey) {
      show(image: cachedImage)
      currentURLString = normalizedURL.absoluteString
      return
    }

    currentURLString = normalizedURL.absoluteString
    imageTask = Task { [weak self] in
      guard let self else { return }
      do {
        let (data, response) = try await URLSession.shared.data(from: normalizedURL)
        guard let httpResponse = response as? HTTPURLResponse,
              (200 ... 299).contains(httpResponse.statusCode),
              let image = UIImage(data: data) else {
          return
        }
        AvatarView.imageCache.setObject(image, forKey: cacheKey)
        guard !Task.isCancelled else { return }
        guard self.currentURLString == normalizedURL.absoluteString else { return }
        self.show(image: image)
      } catch {
        // Keep fallback placeholder.
      }
    }
  }

  private func show(image: UIImage) {
    imageView.image = image
    initialLabel.isHidden = true
    Theme.applySurface(to: self, cornerRadius: bounds.width / 2, fillColor: .clear)
    backgroundColor = .clear
    layer.borderWidth = 0
  }

  private static func normalizedURL(from rawValue: String) -> URL? {
    let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else { return nil }

    if let url = URL(string: trimmed), url.scheme != nil {
      return url
    }

    if trimmed.hasPrefix("http:") {
      let repaired = trimmed.replacingOccurrences(of: "http:", with: "http://")
      return URL(string: repaired)
    }

    if trimmed.hasPrefix("https:") {
      let repaired = trimmed.replacingOccurrences(of: "https:", with: "https://")
      return URL(string: repaired)
    }

    return URL(string: trimmed)
  }

  private static func initials(username: String, firstName: String?, lastName: String?) -> String {
    let firstInitial = firstName?
      .trimmingCharacters(in: .whitespacesAndNewlines)
      .prefix(1) ?? ""
    let lastInitial = lastName?
      .trimmingCharacters(in: .whitespacesAndNewlines)
      .prefix(1) ?? ""
    let combined = "\(firstInitial)\(lastInitial)"
    if combined.isEmpty {
      return String(username.prefix(1)).lowercased()
    }
    return combined.lowercased()
  }
}
