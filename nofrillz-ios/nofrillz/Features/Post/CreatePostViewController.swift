import UIKit

private enum PostNotificationUserInfoKey {
  static let post = "post"
}

private final class ComposerCountAccessoryView: UIView {
  private let accessoryHeight: CGFloat = 28
  private let horizontalInset: CGFloat = 20
  private let minimumLabelWidth: CGFloat = 64

  private let countLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.characterCounter
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.textAlignment = .right
    label.numberOfLines = 1
    label.lineBreakMode = .byClipping
    label.alpha = 0
    return label
  }()

  override var intrinsicContentSize: CGSize {
    CGSize(width: UIView.noIntrinsicMetric, height: accessoryHeight)
  }

  override func sizeThatFits(_ size: CGSize) -> CGSize {
    CGSize(width: size.width, height: accessoryHeight)
  }

  override init(frame: CGRect) {
    super.init(frame: CGRect(x: 0, y: 0, width: UIScreen.main.bounds.width, height: accessoryHeight))
    backgroundColor = .clear
    isOpaque = false
    autoresizingMask = [.flexibleWidth]
    addSubview(countLabel)
  }

  override func layoutSubviews() {
    super.layoutSubviews()
    let labelSize = countLabel.sizeThatFits(bounds.size)
    let maxWidth = max(0, bounds.width - (horizontalInset * 2))
    let labelWidth = min(max(minimumLabelWidth, labelSize.width), maxWidth)
    let labelHeight = min(labelSize.height, bounds.height)
    countLabel.frame = CGRect(
      x: bounds.width - horizontalInset - labelWidth,
      y: round((bounds.height - labelHeight) / 2),
      width: labelWidth,
      height: labelHeight
    )
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  @discardableResult
  func update(
    count: Int,
    maxCount: Int,
    shouldShow: Bool,
    textColor: UIColor,
    labelAlpha: CGFloat
  ) -> Bool {
    countLabel.text = L10n.tr("common.count_of_total", count, maxCount)
    countLabel.textColor = textColor
    countLabel.alpha = shouldShow ? labelAlpha : 0
    return false
  }
}

final class CreatePostViewController: BaseViewController {
  private let environment: AppEnvironment
  var onPostCreated: (() -> Void)?
  private let placeholderRotationInterval: TimeInterval = 10

  private let maxPostLength = 500
  private let counterRevealThreshold = 375
  private let counterStrongContrastThreshold = 450

  private var hasFocusedEditor = false
  private var isPosting = false
  private var placeholderTimer: Timer?
  private var placeholderIndex = 0

  private let editorParagraphStyle: NSParagraphStyle = {
    let style = NSMutableParagraphStyle()
    style.lineSpacing = 6
    return style
  }()

  private lazy var placeholderMessages: [String] = [
    L10n.tr("auth.subtitle.1"),
    L10n.tr("auth.subtitle.2"),
    L10n.tr("auth.subtitle.3")
  ]

  private lazy var editorTextAttributes: [NSAttributedString.Key: Any] = [
    .font: Theme.Typography.composeBody,
    .foregroundColor: Theme.text,
    .paragraphStyle: editorParagraphStyle
  ]

  private let navigationTitleLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.navigationTitle
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.text
    label.textAlignment = .center
    label.text = L10n.tr("create.write_title")
    label.sizeToFit()
    return label
  }()

  private let textView: UITextView = {
    let textView = UITextView()
    textView.backgroundColor = .clear
    textView.textColor = Theme.text
    textView.tintColor = Theme.text
    textView.font = Theme.Typography.composeBody
    textView.adjustsFontForContentSizeCategory = true
    textView.textContainerInset = UIEdgeInsets(top: 24, left: 0, bottom: 36, right: 0)
    textView.textContainer.lineFragmentPadding = 0
    textView.keyboardDismissMode = .interactive
    textView.alwaysBounceVertical = true
    textView.showsVerticalScrollIndicator = false
    textView.autocapitalizationType = .sentences
    textView.autocorrectionType = .yes
    textView.spellCheckingType = .yes
    textView.translatesAutoresizingMaskIntoConstraints = false
    return textView
  }()

  private lazy var placeholderLabel: UILabel = {
    let label = UILabel()
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.placeholderText
    label.numberOfLines = 0
    label.text = L10n.tr("auth.subtitle.1")
    label.textAlignment = .center
    label.isUserInteractionEnabled = false
    label.translatesAutoresizingMaskIntoConstraints = false
    return label
  }()

  private let countAccessoryView = ComposerCountAccessoryView()

  private lazy var postBarButtonItem: UIBarButtonItem = {
    makeNavigationTextBarButtonItem(
      title: L10n.tr("create.post_action"),
      accessibilityLabel: L10n.tr("create.post_action"),
      action: #selector(createPost)
    )
  }()

  private lazy var closeBarButtonItem: UIBarButtonItem = makeNavigationTextBarButtonItem(
    title: L10n.tr("common.cancel"),
    accessibilityLabel: L10n.tr("common.cancel"),
    action: #selector(closeTapped)
  )

  private let activityIndicator: UIActivityIndicatorView = {
    let indicator = UIActivityIndicatorView(style: .medium)
    indicator.color = Theme.text
    indicator.hidesWhenStopped = true
    indicator.translatesAutoresizingMaskIntoConstraints = false
    return indicator
  }()

  private lazy var activityIndicatorContainer: UIView = {
    let container = UIView(frame: CGRect(x: 0, y: 0, width: 44, height: 44))
    container.backgroundColor = .clear
    container.addSubview(activityIndicator)
    NSLayoutConstraint.activate([
      activityIndicator.centerXAnchor.constraint(equalTo: container.centerXAnchor),
      activityIndicator.centerYAnchor.constraint(equalTo: container.centerYAnchor)
    ])
    return container
  }()

  private lazy var activityIndicatorBarButtonItem: UIBarButtonItem = {
    let item = UIBarButtonItem(customView: activityIndicatorContainer)
    Theme.applyEditorialBarButtonStyle(to: item)
    return item
  }()

  init(environment: AppEnvironment) {
    self.environment = environment
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func viewDidLoad() {
    super.viewDidLoad()

    navigationItem.title = nil
    navigationItem.titleView = navigationTitleLabel
    navigationItem.leftBarButtonItem = closeBarButtonItem
    navigationItem.rightBarButtonItem = postBarButtonItem

    textView.delegate = self
    textView.typingAttributes = editorTextAttributes
    textView.inputAccessoryView = countAccessoryView
    textView.inputAssistantItem.leadingBarButtonGroups = []
    textView.inputAssistantItem.trailingBarButtonGroups = []
    textView.addSubview(placeholderLabel)
    view.addSubview(textView)

    NSLayoutConstraint.activate([
      textView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 10),
      textView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 20),
      textView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -20),
      textView.bottomAnchor.constraint(equalTo: view.keyboardLayoutGuide.topAnchor)
    ])

    NSLayoutConstraint.activate([
      placeholderLabel.centerXAnchor.constraint(equalTo: textView.frameLayoutGuide.centerXAnchor),
      placeholderLabel.centerYAnchor.constraint(equalTo: textView.frameLayoutGuide.centerYAnchor),
      placeholderLabel.leadingAnchor.constraint(greaterThanOrEqualTo: textView.frameLayoutGuide.leadingAnchor, constant: 20),
      placeholderLabel.trailingAnchor.constraint(lessThanOrEqualTo: textView.frameLayoutGuide.trailingAnchor, constant: -20),
      placeholderLabel.widthAnchor.constraint(lessThanOrEqualTo: textView.frameLayoutGuide.widthAnchor, constant: -40)
    ])

    updateComposerState(allowAccessoryLayoutRefresh: false)
  }

  override func viewDidAppear(_ animated: Bool) {
    super.viewDidAppear(animated)
    guard !hasFocusedEditor else { return }
    hasFocusedEditor = true
    textView.becomeFirstResponder()
    updatePlaceholderRotationState()
  }

  override func viewWillDisappear(_ animated: Bool) {
    super.viewWillDisappear(animated)
    stopPlaceholderRotation()
  }

  @objc private func closeTapped() {
    dismiss(animated: true)
  }

  @objc private func createPost() {
    let content = textView.text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !content.isEmpty else {
      presentError(L10n.tr("create.empty_error"))
      return
    }
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }

    setPosting(true)

    Task {
      do {
        let post = try await environment.api.createPost(content: content)
        textView.text = ""
        textView.typingAttributes = editorTextAttributes
        updateComposerState(allowAccessoryLayoutRefresh: true)
        NotificationCenter.default.post(
          name: .postCreated,
          object: nil,
          userInfo: [PostNotificationUserInfoKey.post: post]
        )
        Haptics.shared.success()
        onPostCreated?()
      } catch {
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(.publishPost)
      }

      setPosting(false)
    }
  }

  @MainActor
  private func setPosting(_ posting: Bool) {
    isPosting = posting
    textView.isEditable = !posting

    if posting {
      activityIndicator.startAnimating()
      navigationItem.rightBarButtonItem = activityIndicatorBarButtonItem
    } else {
      activityIndicator.stopAnimating()
      navigationItem.rightBarButtonItem = postBarButtonItem
    }

    updateComposerState(allowAccessoryLayoutRefresh: false)
  }

  private func applyEditorAttributesToText() {
    guard !textView.text.isEmpty else {
      textView.typingAttributes = editorTextAttributes
      return
    }

    let selectedRange = textView.selectedRange
    let attributedText = NSMutableAttributedString(string: textView.text, attributes: editorTextAttributes)
    textView.attributedText = attributedText
    textView.selectedRange = selectedRange
    textView.typingAttributes = editorTextAttributes
  }

  private func updatePlaceholderRotationState() {
    let shouldShowPlaceholder = textView.text.isEmpty && view.window != nil
    if shouldShowPlaceholder {
      startPlaceholderRotationIfNeeded()
    } else {
      stopPlaceholderRotation()
    }
  }

  private func startPlaceholderRotationIfNeeded() {
    guard placeholderTimer == nil, !placeholderMessages.isEmpty else { return }
    placeholderIndex = Int.random(in: 0..<placeholderMessages.count)
    placeholderLabel.text = placeholderMessages[placeholderIndex]

    let timer = Timer(timeInterval: placeholderRotationInterval, repeats: true) { [weak self] _ in
      Task { @MainActor [weak self] in
        self?.animateNextPlaceholderMessage()
      }
    }
    timer.fireDate = Date().addingTimeInterval(placeholderRotationInterval)
    RunLoop.main.add(timer, forMode: .common)
    placeholderTimer = timer
  }

  private func stopPlaceholderRotation() {
    placeholderTimer?.invalidate()
    placeholderTimer = nil
  }

  private func animateNextPlaceholderMessage() {
    guard !placeholderMessages.isEmpty, textView.text.isEmpty else { return }
    placeholderIndex = (placeholderIndex + 1) % placeholderMessages.count
    let nextMessage = placeholderMessages[placeholderIndex]

    UIView.transition(
      with: placeholderLabel,
      duration: 1.0,
      options: [.transitionCrossDissolve, .allowUserInteraction]
    ) {
      self.placeholderLabel.text = nextMessage
    }
  }

  private func updateComposerState(allowAccessoryLayoutRefresh: Bool) {
    let trimmedContent = textView.text.trimmingCharacters(in: .whitespacesAndNewlines)
    let characterCount = textView.text.count
    let canSubmit = !isPosting && !trimmedContent.isEmpty && characterCount <= maxPostLength
    postBarButtonItem.isEnabled = canSubmit
    placeholderLabel.isHidden = !textView.text.isEmpty
    updatePlaceholderRotationState()

    let shouldShowCounter = characterCount > 0
    let counterColor: UIColor
    let counterAlpha: CGFloat
    if characterCount > maxPostLength {
      counterColor = Theme.red.withAlphaComponent(0.85)
      counterAlpha = 0.96
    } else if characterCount >= counterStrongContrastThreshold {
      counterColor = Theme.text.withAlphaComponent(0.58)
      counterAlpha = 0.78
    } else if characterCount >= counterRevealThreshold {
      counterColor = Theme.secondaryText.withAlphaComponent(0.82)
      counterAlpha = 0.56
    } else {
      counterColor = Theme.secondaryText.withAlphaComponent(0.82)
      counterAlpha = 0.5
    }

    _ = countAccessoryView.update(
      count: characterCount,
      maxCount: maxPostLength,
      shouldShow: shouldShowCounter,
      textColor: counterColor,
      labelAlpha: counterAlpha
    )
  }
}

extension CreatePostViewController: UITextViewDelegate {
  func textViewDidChange(_ textView: UITextView) {
    applyEditorAttributesToText()
    updateComposerState(allowAccessoryLayoutRefresh: true)
  }
}
