import UIKit

final class NotificationsSettingsViewController: BaseViewController {
  private let horizontalContentInset: CGFloat = 16

  private final class ToggleRowView: UIView {
    let preference: Preference
    let titleLabel = UILabel()
    let subtitleLabel = UILabel()
    let toggle = UISwitch()

    init(preference: Preference, isOn: Bool) {
      self.preference = preference
      super.init(frame: .zero)
      translatesAutoresizingMaskIntoConstraints = false
      heightAnchor.constraint(greaterThanOrEqualToConstant: 76).isActive = true

      titleLabel.font = Theme.Typography.settingsRowTitle
      titleLabel.adjustsFontForContentSizeCategory = true
      titleLabel.textColor = Theme.text
      titleLabel.text = preference.title

      subtitleLabel.font = Theme.Typography.settingsDescription
      subtitleLabel.adjustsFontForContentSizeCategory = true
      subtitleLabel.textColor = Theme.secondaryText
      subtitleLabel.text = preference.subtitle
      subtitleLabel.numberOfLines = 0

      let labels = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel])
      labels.axis = .vertical
      labels.alignment = .leading
      labels.spacing = 3
      labels.translatesAutoresizingMaskIntoConstraints = false

      toggle.onTintColor = Theme.green
      toggle.isOn = isOn
      toggle.translatesAutoresizingMaskIntoConstraints = false

      addSubview(labels)
      addSubview(toggle)

      NSLayoutConstraint.activate([
        labels.topAnchor.constraint(equalTo: topAnchor, constant: 18),
        labels.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -18),
        labels.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),

        toggle.leadingAnchor.constraint(greaterThanOrEqualTo: labels.trailingAnchor, constant: 12),
        toggle.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
        toggle.centerYAnchor.constraint(equalTo: centerYAnchor)
      ])
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
      fatalError("init(coder:) has not been implemented")
    }

    func setContentEnabled(_ isEnabled: Bool) {
      toggle.isEnabled = isEnabled
      titleLabel.textColor = isEnabled ? Theme.text : Theme.secondaryText
      subtitleLabel.textColor = isEnabled ? Theme.secondaryText : Theme.secondaryText.withAlphaComponent(0.7)
      alpha = isEnabled ? 1 : 0.5
    }
  }

  private struct Preference {
    let title: String
    let subtitle: String
    let key: String
  }

  private enum Copy {
    static let title = L10n.tr("notifications.title")
    static let note = L10n.tr("notifications.note")
    static let previewTitle = L10n.tr("notifications.preview.title")
    static let previewSubtitle = L10n.tr("notifications.preview.subtitle")
    static let master = Preference(
      title: L10n.tr("notifications.master.title"),
      subtitle: L10n.tr("notifications.master.subtitle"),
      key: "nofrillz.notifications.master"
    )
    static let likes = Preference(
      title: L10n.tr("notifications.likes.title"),
      subtitle: L10n.tr("notifications.likes.subtitle"),
      key: "nofrillz.notifications.likes"
    )
    static let follows = Preference(
      title: L10n.tr("notifications.follows.title"),
      subtitle: L10n.tr("notifications.follows.subtitle"),
      key: "nofrillz.notifications.follows"
    )
    static let replies = Preference(
      title: L10n.tr("notifications.replies.title"),
      subtitle: L10n.tr("notifications.replies.subtitle"),
      key: "nofrillz.notifications.replies"
    )
  }

  private let defaults = UserDefaults.standard
  private var masterRow: ToggleRowView?
  private var dependentRows: [ToggleRowView] = []

  private lazy var previewBannerButton = makeActionRow(
    title: Copy.previewTitle,
    subtitle: Copy.previewSubtitle
  )

  private let scrollView: UIScrollView = {
    let scrollView = UIScrollView()
    scrollView.translatesAutoresizingMaskIntoConstraints = false
    scrollView.alwaysBounceVertical = true
    return scrollView
  }()

  private let contentStack: UIStackView = {
    let stack = UIStackView()
    stack.axis = .vertical
    stack.spacing = 0
    stack.translatesAutoresizingMaskIntoConstraints = false
    return stack
  }()

  override func viewDidLoad() {
    super.viewDidLoad()
    configureNavigationTitle()
    setupView()
    wireActions()
  }

  private func configureNavigationTitle() {
    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.navigationTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = Copy.title
    navigationItem.titleView = titleLabel
  }

  private func setupView() {
    view.addSubview(scrollView)
    scrollView.addSubview(contentStack)

    NSLayoutConstraint.activate([
      scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
      scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
      scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
      scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),

      contentStack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 20),
      contentStack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -28),
      contentStack.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor),
      contentStack.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor)
    ])

    contentStack.addArrangedSubview(makeNoteSection())
    contentStack.addArrangedSubview(makeSpacer(height: 24))
    contentStack.addArrangedSubview(makePreferencesSection())
    contentStack.addArrangedSubview(makeSpacer(height: 24))
    contentStack.addArrangedSubview(makePreviewSection())
  }

  private func makeNoteSection() -> UIView {
    let container = UIView()
    let label = UILabel()
    label.font = Theme.Typography.helperText
    label.adjustsFontForContentSizeCategory = true
    label.textColor = Theme.secondaryText
    label.numberOfLines = 0
    label.text = Copy.note
    label.translatesAutoresizingMaskIntoConstraints = false
    container.addSubview(label)

    NSLayoutConstraint.activate([
      label.topAnchor.constraint(equalTo: container.topAnchor),
      label.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: horizontalContentInset),
      label.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -horizontalContentInset),
      label.bottomAnchor.constraint(equalTo: container.bottomAnchor)
    ])

    return container
  }

  private func makePreferencesSection() -> UIView {
    let masterRow = makeToggleRow(for: Copy.master)
    let likesRow = makeToggleRow(for: Copy.likes)
    let followsRow = makeToggleRow(for: Copy.follows)
    let repliesRow = makeToggleRow(for: Copy.replies)

    self.masterRow = masterRow
    dependentRows = [likesRow, followsRow, repliesRow]

    let stack = UIStackView(arrangedSubviews: [
      masterRow,
      makeSeparator(),
      likesRow,
      makeSeparator(),
      followsRow,
      makeSeparator(),
      repliesRow,
      makeSeparator()
    ])
    stack.axis = .vertical
    stack.spacing = 0
    updateDependentRowsAppearance()
    return stack
  }

  private func makePreviewSection() -> UIView {
    let stack = UIStackView(arrangedSubviews: [
      previewBannerButton,
      makeSeparator()
    ])
    stack.axis = .vertical
    stack.spacing = 0
    return stack
  }

  private func makeToggleRow(for preference: Preference) -> ToggleRowView {
    let row = ToggleRowView(
      preference: preference,
      isOn: defaults.object(forKey: preference.key) as? Bool ?? defaultValue(for: preference.key)
    )
    row.toggle.addAction(UIAction { [weak self, weak row] _ in
      guard let self, let row else { return }
      defaults.set(row.toggle.isOn, forKey: preference.key)
      if preference.key == Copy.master.key {
        updateDependentRowsAppearance()
      }
    }, for: .valueChanged)
    return row
  }

  private func updateDependentRowsAppearance() {
    let isPushEnabled = masterRow?.toggle.isOn ?? defaultValue(for: Copy.master.key)
    dependentRows.forEach { row in
      row.setContentEnabled(isPushEnabled)
    }
  }

  private func makeActionRow(title: String, subtitle: String?) -> UIButton {
    let button = UIButton(type: .custom)
    button.contentHorizontalAlignment = .fill
    button.backgroundColor = .clear
    button.translatesAutoresizingMaskIntoConstraints = false
    button.heightAnchor.constraint(greaterThanOrEqualToConstant: 76).isActive = true

    let titleLabel = UILabel()
    titleLabel.font = Theme.Typography.settingsRowTitle
    titleLabel.adjustsFontForContentSizeCategory = true
    titleLabel.textColor = Theme.text
    titleLabel.text = title
    titleLabel.numberOfLines = 1

    let subtitleLabel = UILabel()
    subtitleLabel.font = Theme.Typography.settingsDescription
    subtitleLabel.adjustsFontForContentSizeCategory = true
    subtitleLabel.textColor = Theme.secondaryText
    subtitleLabel.text = subtitle
    subtitleLabel.numberOfLines = 0

    let labelsStack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel])
    labelsStack.axis = .vertical
    labelsStack.alignment = .leading
    labelsStack.spacing = 3
    labelsStack.isUserInteractionEnabled = false

    let chevronView = UIImageView(image: AppIcon.chevronRight.image)
    chevronView.tintColor = Theme.secondaryText
    chevronView.contentMode = .scaleAspectFit
    chevronView.translatesAutoresizingMaskIntoConstraints = false
    NSLayoutConstraint.activate([
      chevronView.widthAnchor.constraint(equalToConstant: 32),
      chevronView.heightAnchor.constraint(equalToConstant: 32)
    ])

    let row = UIStackView(arrangedSubviews: [labelsStack, chevronView])
    row.axis = .horizontal
    row.alignment = .center
    row.spacing = 12
    row.isUserInteractionEnabled = false
    row.translatesAutoresizingMaskIntoConstraints = false
    button.addSubview(row)

    NSLayoutConstraint.activate([
      row.topAnchor.constraint(equalTo: button.topAnchor, constant: 18),
      row.bottomAnchor.constraint(equalTo: button.bottomAnchor, constant: -18),
      row.leadingAnchor.constraint(equalTo: button.leadingAnchor, constant: horizontalContentInset),
      row.trailingAnchor.constraint(equalTo: button.trailingAnchor, constant: -horizontalContentInset)
    ])

    return button
  }

  private func makeSeparator() -> UIView {
    let view = UIView()
    view.backgroundColor = Theme.separator
    view.translatesAutoresizingMaskIntoConstraints = false
    view.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
    return view
  }

  private func makeSpacer(height: CGFloat) -> UIView {
    let view = UIView()
    view.translatesAutoresizingMaskIntoConstraints = false
    view.heightAnchor.constraint(equalToConstant: height).isActive = true
    return view
  }

  private func defaultValue(for key: String) -> Bool {
    switch key {
    case Copy.master.key:
      return true
    case Copy.likes.key, Copy.follows.key, Copy.replies.key:
      return true
    default:
      return false
    }
  }

  private func wireActions() {
    previewBannerButton.addTarget(self, action: #selector(previewBannerTapped), for: .touchUpInside)
  }

  @objc private func previewBannerTapped() {
    PushNotificationManager.shared.showPreviewBanner()
  }
}
