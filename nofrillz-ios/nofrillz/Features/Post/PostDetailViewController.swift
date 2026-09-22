import UIKit

final class PostDetailViewController: BaseViewController {
  private let environment: AppEnvironment
  private let initialPost: Post
  private var currentPost: Post
  private var authorUser: User?
  private var isLikeRequestInFlight = false

  private let navigationTitleContainer = UIView(frame: CGRect(x: 0, y: 0, width: 220, height: 32))
  private let navigationTitleLabel = UILabel()
  private let navigationAuthorLabel = UILabel()

  private let scrollView = UIScrollView()
  private let contentView = UIView()
  private let mainStack = UIStackView()

  private let authorHeaderView = UIView()
  private let authorAvatarView = AvatarView()
  private let authorNameLabel = UILabel()
  private let authorSubtitleLabel = UILabel()

  private let heartButton = UIButton(type: .system)
  private let shareButton = UIButton(type: .system)
  private let bookmarkButton = UIButton(type: .system)
  private let primaryActionsRow = UIStackView()
  private let actionsSpacerView = UIView()
  private let actionsRow = UIStackView()
  private let bodyTextView = UITextView()

  init(environment: AppEnvironment, post: Post) {
    self.environment = environment
    self.initialPost = post
    currentPost = environment.bookmarkStore.displayedPost(from: post)
    super.init(nibName: nil, bundle: nil)
  }

  @available(*, unavailable)
  required init?(coder: NSCoder) {
    fatalError("init(coder:) has not been implemented")
  }

  override func viewDidLoad() {
    super.viewDidLoad()
    title = L10n.tr("post.title")
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleBookmarkStateChange(_:)),
      name: .bookmarkStateDidChange,
      object: nil
    )
    configureNavigationTitle()
    configureNavigationMenu()
    setupView()
    authorUser = fallbackAuthorUser(from: initialPost)
    apply(post: currentPost)
    refreshPost()
  }

  override func viewDidLayoutSubviews() {
    super.viewDidLayoutSubviews()
    updateNavigationTitleVisibility()
  }

  private func configureNavigationTitle() {
    navigationTitleLabel.font = Theme.Typography.navigationTitle
    navigationTitleLabel.adjustsFontForContentSizeCategory = true
    navigationTitleLabel.textColor = Theme.text
    navigationTitleLabel.textAlignment = .center
    navigationTitleLabel.text = L10n.tr("post.title")
    navigationTitleLabel.translatesAutoresizingMaskIntoConstraints = false

    navigationAuthorLabel.font = Theme.Typography.editorialHeadline
    navigationAuthorLabel.adjustsFontForContentSizeCategory = true
    navigationAuthorLabel.textColor = Theme.text
    navigationAuthorLabel.textAlignment = .center
    navigationAuthorLabel.lineBreakMode = .byTruncatingTail
    navigationAuthorLabel.numberOfLines = 1
    navigationAuthorLabel.alpha = 0
    navigationAuthorLabel.translatesAutoresizingMaskIntoConstraints = false

    navigationTitleContainer.addSubview(navigationTitleLabel)
    navigationTitleContainer.addSubview(navigationAuthorLabel)

    NSLayoutConstraint.activate([
      navigationTitleLabel.centerXAnchor.constraint(equalTo: navigationTitleContainer.centerXAnchor),
      navigationTitleLabel.centerYAnchor.constraint(equalTo: navigationTitleContainer.centerYAnchor),
      navigationTitleLabel.leadingAnchor.constraint(greaterThanOrEqualTo: navigationTitleContainer.leadingAnchor),
      navigationTitleLabel.trailingAnchor.constraint(lessThanOrEqualTo: navigationTitleContainer.trailingAnchor),

      navigationAuthorLabel.centerXAnchor.constraint(equalTo: navigationTitleContainer.centerXAnchor),
      navigationAuthorLabel.centerYAnchor.constraint(equalTo: navigationTitleContainer.centerYAnchor),
      navigationAuthorLabel.leadingAnchor.constraint(greaterThanOrEqualTo: navigationTitleContainer.leadingAnchor),
      navigationAuthorLabel.trailingAnchor.constraint(lessThanOrEqualTo: navigationTitleContainer.trailingAnchor)
    ])

    navigationItem.titleView = navigationTitleContainer
  }

  private func configureNavigationMenu() {
    navigationItem.rightBarButtonItem = makeNavigationMenuBarButtonItem(
      image: AppIcon.more.image,
      accessibilityLabel: L10n.tr("common.more"),
      menu: UIMenu(children: [
        UIAction(title: L10n.tr("common.report"), attributes: .destructive) { _ in }
      ]),
      iconSize: 28,
      buttonSize: 44
    )
  }

  private func setupView() {
    scrollView.translatesAutoresizingMaskIntoConstraints = false
    scrollView.alwaysBounceVertical = true
    scrollView.delegate = self

    contentView.translatesAutoresizingMaskIntoConstraints = false

    mainStack.axis = .vertical
    mainStack.spacing = 24
    mainStack.translatesAutoresizingMaskIntoConstraints = false

    configureAuthorHeader()
    configureActionsRow()
    configureBodyTextView()

    mainStack.addArrangedSubview(authorHeaderView)
    mainStack.addArrangedSubview(bodyTextView)
    mainStack.addArrangedSubview(actionsRow)
    mainStack.setCustomSpacing(18, after: authorHeaderView)
    mainStack.setCustomSpacing(18, after: bodyTextView)

    view.addSubview(scrollView)
    scrollView.addSubview(contentView)
    contentView.addSubview(mainStack)

    NSLayoutConstraint.activate([
      scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),

      contentView.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor),
      contentView.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor),
      contentView.leadingAnchor.constraint(equalTo: scrollView.contentLayoutGuide.leadingAnchor),
      contentView.trailingAnchor.constraint(equalTo: scrollView.contentLayoutGuide.trailingAnchor),
      contentView.widthAnchor.constraint(equalTo: scrollView.frameLayoutGuide.widthAnchor),

      mainStack.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 20),
      mainStack.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 20),
      mainStack.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -20),
      mainStack.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -32)
    ])
  }

  private func configureAuthorHeader() {
    authorHeaderView.translatesAutoresizingMaskIntoConstraints = false
    authorHeaderView.isUserInteractionEnabled = true
    authorHeaderView.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(authorTapped)))

    authorAvatarView.translatesAutoresizingMaskIntoConstraints = false
    authorAvatarView.widthAnchor.constraint(equalToConstant: 52).isActive = true
    authorAvatarView.heightAnchor.constraint(equalToConstant: 52).isActive = true

    authorNameLabel.font = Theme.Typography.editorialHeadline
    authorNameLabel.adjustsFontForContentSizeCategory = true
    authorNameLabel.textColor = Theme.text
    authorNameLabel.numberOfLines = 1
    authorNameLabel.lineBreakMode = .byTruncatingTail

    authorSubtitleLabel.font = Theme.Typography.metadata
    authorSubtitleLabel.adjustsFontForContentSizeCategory = true
    authorSubtitleLabel.textColor = Theme.secondaryText
    authorSubtitleLabel.numberOfLines = 0

    let authorTextStack = UIStackView(arrangedSubviews: [authorNameLabel, authorSubtitleLabel])
    authorTextStack.axis = .vertical
    authorTextStack.spacing = 4
    authorTextStack.alignment = .leading
    authorTextStack.translatesAutoresizingMaskIntoConstraints = false

    let authorRow = UIStackView(arrangedSubviews: [authorAvatarView, authorTextStack])
    authorRow.axis = .horizontal
    authorRow.alignment = .center
    authorRow.spacing = 12
    authorRow.translatesAutoresizingMaskIntoConstraints = false

    authorHeaderView.addSubview(authorRow)

    NSLayoutConstraint.activate([
      authorRow.topAnchor.constraint(equalTo: authorHeaderView.topAnchor),
      authorRow.bottomAnchor.constraint(equalTo: authorHeaderView.bottomAnchor),
      authorRow.leadingAnchor.constraint(equalTo: authorHeaderView.leadingAnchor),
      authorRow.trailingAnchor.constraint(equalTo: authorHeaderView.trailingAnchor)
    ])
  }

  private func configureActionsRow() {
    configureActionButton(heartButton, image: AppIcon.heart.image, accessibilityLabel: "Respect", action: #selector(toggleLike))
    configureActionButton(shareButton, image: AppIcon.share.image, accessibilityLabel: L10n.tr("common.share"), action: #selector(shareTapped))
    configureActionButton(
      bookmarkButton,
      image: AppIcon.bookmark.image,
      accessibilityLabel: "Save to Bookmarks",
      action: #selector(toggleBookmark)
    )

    primaryActionsRow.axis = .horizontal
    primaryActionsRow.alignment = .center
    primaryActionsRow.spacing = Theme.postActionSpacing
    primaryActionsRow.addArrangedSubview(heartButton)
    primaryActionsRow.addArrangedSubview(shareButton)

    actionsRow.axis = .horizontal
    actionsRow.alignment = .center
    actionsRow.spacing = Theme.postActionSpacing
    actionsRow.translatesAutoresizingMaskIntoConstraints = false
    actionsSpacerView.setContentHuggingPriority(.defaultLow, for: .horizontal)
    actionsSpacerView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    actionsRow.addArrangedSubview(primaryActionsRow)
    actionsRow.addArrangedSubview(actionsSpacerView)
    actionsRow.addArrangedSubview(bookmarkButton)
  }

  private func configureActionButton(
    _ button: UIButton,
    image: UIImage?,
    accessibilityLabel: String,
    action: Selector
  ) {
    button.setImage(image, for: .normal)
    button.tintColor = Theme.text
    button.imageView?.contentMode = .scaleAspectFit
    button.accessibilityLabel = accessibilityLabel
    button.translatesAutoresizingMaskIntoConstraints = false
    button.addTarget(self, action: action, for: .touchUpInside)
    NSLayoutConstraint.activate([
      button.widthAnchor.constraint(equalToConstant: Theme.postActionIconSize),
      button.heightAnchor.constraint(equalToConstant: Theme.postActionIconSize)
    ])
  }

  private func configureBodyTextView() {
    bodyTextView.font = Theme.Typography.postBody
    bodyTextView.adjustsFontForContentSizeCategory = true
    bodyTextView.textColor = Theme.text
    bodyTextView.backgroundColor = .clear
    bodyTextView.isEditable = false
    bodyTextView.isSelectable = true
    bodyTextView.isScrollEnabled = false
    bodyTextView.textContainerInset = .zero
    bodyTextView.textContainer.lineFragmentPadding = 0
    bodyTextView.dataDetectorTypes = [.link]
    bodyTextView.linkTextAttributes = [
      .underlineStyle: NSUnderlineStyle.single.rawValue,
      .foregroundColor: Theme.text
    ]
    bodyTextView.delegate = self
    bodyTextView.setContentCompressionResistancePriority(.required, for: .vertical)
  }

  private func apply(post: Post) {
    currentPost = post
    navigationAuthorLabel.text = post.authorDisplayName
    authorNameLabel.text = post.authorDisplayName
    authorSubtitleLabel.text = makeAuthorSubtitle(for: post)

    authorAvatarView.configure(
      username: post.authorUsername,
      firstName: post.authorFirstName,
      lastName: post.authorLastName,
      avatarURL: post.authorAvatarURL
    )

    bodyTextView.attributedText = makeBodyAttributedText(post.content)
    applyLikeButton(isLiked: post.liked)
    applyBookmarkButton(
      isBookmarked: post.isBookmarked,
      isMutationInFlight: environment.bookmarkStore.isMutationInFlight(for: post.id)
    )

    let canShare = !(post.url?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
    shareButton.isEnabled = canShare
    shareButton.alpha = canShare ? 1 : 0.35

    if authorUser?.id != post.authorID {
      authorUser = fallbackAuthorUser(from: post)
    }

    updateNavigationTitleVisibility()
  }

  private func refreshPost() {
    Task {
      do {
        let latest = try await environment.api.fetchPost(id: initialPost.id)
        let resolved = environment.bookmarkStore.absorbServerPosts([latest]).first ?? latest
        apply(post: resolved)
      } catch {
        // Keep initial post visible when detail fetch fails.
      }
    }
  }

  private func fallbackAuthorUser(from post: Post) -> User {
    if let currentUser = environment.sessionStore.currentUser, currentUser.id == post.authorID {
      return currentUser
    }

    return User(
      id: post.authorID,
      username: post.authorUsername,
      firstName: post.authorFirstName,
      lastName: post.authorLastName,
      email: nil,
      about: nil,
      avatarURL: post.authorAvatarURL,
      joinedAt: nil,
      followerCount: 0,
      followingCount: 0,
      postCount: 0
    )
  }

  private func makeAuthorSubtitle(for post: Post) -> String {
    var parts = ["@\(post.authorUsername)"]

    if let createdAt = post.createdAt {
      parts.append(PostDateDisplay.shortString(from: createdAt))
    }

    if let readingTimeText = post.displayEstimatedReadingTimeText {
      parts.append(readingTimeText)
    }

    if post.isAI {
      parts.append(L10n.tr("common.ai_label"))
    }

    return parts.joined(separator: " • ")
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

  private func applyLikeButton(isLiked: Bool) {
    heartButton.setImage(isLiked ? AppIcon.heartFilled.image : AppIcon.heart.image, for: .normal)
    heartButton.tintColor = isLiked ? Theme.likedAccent : Theme.text
  }

  private func applyBookmarkButton(isBookmarked: Bool, isMutationInFlight: Bool) {
    bookmarkButton.setImage(isBookmarked ? AppIcon.bookmarkFilled.image : AppIcon.bookmark.image, for: .normal)
    bookmarkButton.tintColor = isBookmarked ? Theme.bookmarkedAccent : Theme.text
    bookmarkButton.isEnabled = !isMutationInFlight
    bookmarkButton.alpha = isMutationInFlight ? 0.45 : 1
    bookmarkButton.accessibilityLabel = isBookmarked ? "Remove from Bookmarks" : "Save to Bookmarks"
    bookmarkButton.accessibilityValue = isBookmarked ? "Saved" : "Not saved"
  }

  private func animateLikeTransition(duration: TimeInterval = 0.18) {
    heartButton.transform = .identity
    heartButton.alpha = 1
    UIView.animate(withDuration: duration / 2, delay: 0, options: [.curveEaseOut, .beginFromCurrentState]) {
      self.heartButton.transform = CGAffineTransform(scaleX: 1.04, y: 1.04)
      self.heartButton.alpha = 0.82
    } completion: { _ in
      UIView.animate(withDuration: duration / 2, delay: 0, options: [.curveEaseIn, .beginFromCurrentState]) {
        self.heartButton.transform = .identity
        self.heartButton.alpha = 1
      }
    }
  }

  private func updateNavigationTitleVisibility() {
    guard isViewLoaded else { return }

    let authorFrame = authorHeaderView.convert(authorHeaderView.bounds, to: view)
    let thresholdY = view.safeAreaInsets.top + 6
    let fadeDistance: CGFloat = 28
    let progress = max(0, min(1, (thresholdY - authorFrame.maxY) / fadeDistance))
    navigationTitleLabel.alpha = 1 - progress
    navigationAuthorLabel.alpha = progress
  }

  @objc private func toggleLike() {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }
    guard !isLikeRequestInFlight else { return }
    let originalPost = currentPost
    let desiredLikedState = !currentPost.liked
    isLikeRequestInFlight = true
    apply(post: currentPost.withLiked(desiredLikedState))
    Task {
      do {
        if desiredLikedState {
          try await environment.api.likePost(id: currentPost.id)
        } else {
          try await environment.api.unlikePost(id: currentPost.id)
        }
        isLikeRequestInFlight = false

        if currentPost.liked == desiredLikedState {
          animateLikeTransition()
          Haptics.shared.softImpact()
        }
      } catch {
        isLikeRequestInFlight = false
        guard currentPost.liked == desiredLikedState else { return }
        apply(post: originalPost)
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(
          desiredLikedState ? .respect : .removeRespect
        )
      }
    }
  }

  @objc private func toggleBookmark() {
    if environment.connectivityMonitor.isOffline {
      Haptics.shared.error()
      environment.feedbackCenter.showOfflineToastIfNeeded()
      return
    }
    let shouldBookmark = !currentPost.isBookmarked

    Task {
      do {
        let updatedPost = try await environment.bookmarkStore.setBookmarked(shouldBookmark, for: currentPost)
        apply(post: updatedPost)
        Haptics.shared.lightImpact()
      } catch {
        Haptics.shared.error()
        environment.feedbackCenter.showRecoverableActionFailure(
          shouldBookmark ? .saveBookmark : .removeBookmark
        )
      }
    }
  }

  @objc private func handleBookmarkStateChange(_ notification: Notification) {
    guard let change = notification.bookmarkStateChange else { return }
    guard change.post.id == currentPost.id else { return }
    apply(post: change.post)
  }

  @objc private func authorTapped() {
    if let existingProfileViewController = navigationController?.viewControllers
      .dropLast()
      .reversed()
      .compactMap({ $0 as? UserProfileViewController })
      .first(where: { $0.resolvedUserID == currentPost.authorID }) {
      navigationController?.popToViewController(existingProfileViewController, animated: true)
      return
    }

    let initialUser = authorUser ?? fallbackAuthorUser(from: currentPost)
    navigationController?.pushViewController(
      UserProfileViewController(
        environment: environment,
        userID: currentPost.authorID,
        initialUser: initialUser
      ),
      animated: true
    )
  }

  @objc private func shareTapped() {
    guard let url = environment.api.resolveURL(path: currentPost.url) else { return }
    presentShareSheet(items: [url], sourceView: shareButton)
  }
}

extension PostDetailViewController: UITextViewDelegate {
  func textView(
    _ textView: UITextView,
    shouldInteractWith url: URL,
    in characterRange: NSRange,
    interaction: UITextItemInteraction
  ) -> Bool {
    presentWebBrowser(url: url)
    return false
  }
}

extension PostDetailViewController: UIScrollViewDelegate {
  func scrollViewDidScroll(_ scrollView: UIScrollView) {
    updateNavigationTitleVisibility()
  }
}
