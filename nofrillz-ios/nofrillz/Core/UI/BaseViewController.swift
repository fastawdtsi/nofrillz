import UIKit
import SafariServices

@MainActor
class BaseViewController: UIViewController {
  override func viewDidLoad() {
    super.viewDidLoad()
    view.backgroundColor = Theme.background
    navigationItem.largeTitleDisplayMode = .never
    configureCustomBackButtonIfNeeded()
  }

  override func viewWillAppear(_ animated: Bool) {
    super.viewWillAppear(animated)
    configureCustomBackButtonIfNeeded()
  }

  func presentError(_ message: String) {
    let alert = UIAlertController(title: L10n.tr("common.error"), message: message, preferredStyle: .alert)
    alert.addAction(UIAlertAction(title: L10n.tr("common.ok"), style: .default))
    present(alert, animated: true)
  }

  var isInVisibleHierarchy: Bool {
    isViewLoaded && view.window != nil
  }

  func presentWebBrowser(url: URL) {
    let browser = SFSafariViewController(url: url)
    browser.dismissButtonStyle = .close
    present(browser, animated: true)
  }

  func presentShareSheet(items: [Any], sourceView: UIView? = nil) {
    Haptics.shared.lightImpact()
    let controller = UIActivityViewController(activityItems: items, applicationActivities: nil)
    if let popover = controller.popoverPresentationController {
      let anchorView: UIView = sourceView ?? (viewIfLoaded ?? view)
      popover.sourceView = anchorView
      popover.sourceRect = CGRect(
        x: anchorView.bounds.midX,
        y: anchorView.bounds.midY,
        width: 0,
        height: 0
      )
      popover.permittedArrowDirections = sourceView == nil ? [] : .any
    }
    present(controller, animated: true)
  }

  func makeNavigationIconBarButtonItem(
    image: UIImage?,
    accessibilityLabel: String,
    action: Selector,
    imageTransform: CGAffineTransform = .identity,
    iconSize: CGFloat = 28,
    buttonSize: CGFloat = 44
  ) -> UIBarButtonItem {
    let container = UIView(frame: CGRect(x: 0, y: 0, width: buttonSize, height: buttonSize))
    container.backgroundColor = .clear

    let button = UIButton(type: .custom)
    button.frame = container.bounds
    button.autoresizingMask = [.flexibleWidth, .flexibleHeight]
    button.accessibilityLabel = accessibilityLabel
    button.addTarget(self, action: action, for: .touchUpInside)
    container.addSubview(button)

    let imageView = UIImageView(image: image?.withRenderingMode(.alwaysTemplate))
    imageView.translatesAutoresizingMaskIntoConstraints = false
    imageView.tintColor = Theme.text
    imageView.contentMode = .scaleAspectFit
    imageView.isUserInteractionEnabled = false
    imageView.transform = imageTransform
    button.addSubview(imageView)

    NSLayoutConstraint.activate([
      imageView.widthAnchor.constraint(equalToConstant: iconSize),
      imageView.heightAnchor.constraint(equalToConstant: iconSize),
      imageView.centerXAnchor.constraint(equalTo: button.centerXAnchor),
      imageView.centerYAnchor.constraint(equalTo: button.centerYAnchor)
    ])

    let item = UIBarButtonItem(customView: container)
    Theme.applyEditorialBarButtonStyle(to: item)
    return item
  }

  func makeNavigationMenuBarButtonItem(
    image: UIImage?,
    accessibilityLabel: String,
    menu: UIMenu,
    iconSize: CGFloat = 28,
    buttonSize: CGFloat = 44
  ) -> UIBarButtonItem {
    let container = UIView(frame: CGRect(x: 0, y: 0, width: buttonSize, height: buttonSize))
    container.backgroundColor = .clear

    let button = UIButton(type: .custom)
    button.frame = container.bounds
    button.autoresizingMask = [.flexibleWidth, .flexibleHeight]
    button.accessibilityLabel = accessibilityLabel
    button.showsMenuAsPrimaryAction = true
    button.menu = menu
    button.addTarget(self, action: #selector(menuPresentationTouchedDown), for: .touchDown)
    container.addSubview(button)

    let imageView = UIImageView(image: image?.withRenderingMode(.alwaysTemplate))
    imageView.translatesAutoresizingMaskIntoConstraints = false
    imageView.tintColor = Theme.text
    imageView.contentMode = .scaleAspectFit
    imageView.isUserInteractionEnabled = false
    button.addSubview(imageView)

    NSLayoutConstraint.activate([
      imageView.widthAnchor.constraint(equalToConstant: iconSize),
      imageView.heightAnchor.constraint(equalToConstant: iconSize),
      imageView.centerXAnchor.constraint(equalTo: button.centerXAnchor),
      imageView.centerYAnchor.constraint(equalTo: button.centerYAnchor)
    ])

    let item = UIBarButtonItem(customView: container)
    Theme.applyEditorialBarButtonStyle(to: item)
    return item
  }

  func makeNavigationTextBarButtonItem(
    title: String,
    accessibilityLabel: String,
    action: Selector
  ) -> UIBarButtonItem {
    let item = UIBarButtonItem(title: title, style: .plain, target: self, action: action)
    item.accessibilityLabel = accessibilityLabel
    item.setTitleTextAttributes(Theme.Typography.navigationActionTextAttributes, for: .normal)
    item.setTitleTextAttributes(Theme.Typography.disabledNavigationActionTextAttributes, for: .disabled)
    Theme.applyEditorialBarButtonStyle(to: item)
    return item
  }

  func makeNavigationBackBarButtonItem(
    accessibilityLabel: String,
    action: Selector
  ) -> UIBarButtonItem {
    let button = UIButton(type: .system)
    button.accessibilityLabel = accessibilityLabel
    button.addTarget(self, action: action, for: .touchUpInside)
    button.setImage(AppIcon.chevronLeft.image, for: .normal)
    button.tintColor = Theme.text
    button.contentHorizontalAlignment = .leading
    button.contentVerticalAlignment = .center
    button.semanticContentAttribute = .forceLeftToRight
    button.sizeToFit()

    let buttonHeight: CGFloat = 44
    let buttonWidth: CGFloat = 44
    let container = UIView(frame: CGRect(x: 0, y: 0, width: buttonWidth, height: buttonHeight))
    container.backgroundColor = .clear

    button.frame = CGRect(
      x: 0,
      y: round((buttonHeight - button.bounds.height) / 2),
      width: buttonWidth,
      height: button.bounds.height
    )
    container.addSubview(button)

    let item = UIBarButtonItem(customView: container)
    Theme.applyEditorialBarButtonStyle(to: item)
    return item
  }

  private func configureCustomBackButtonIfNeeded() {
    guard navigationController?.viewControllers.first !== self else { return }
    guard navigationItem.leftBarButtonItem == nil else { return }

    navigationItem.hidesBackButton = true
    navigationItem.leftItemsSupplementBackButton = false

    navigationItem.leftBarButtonItem = makeNavigationBackBarButtonItem(
      accessibilityLabel: L10n.tr("common.back"),
      action: #selector(backButtonTapped)
    )
  }

  @objc private func backButtonTapped() {
    navigationController?.popViewController(animated: true)
  }

  @objc private func menuPresentationTouchedDown() {
    Haptics.shared.lightImpact()
  }
}
