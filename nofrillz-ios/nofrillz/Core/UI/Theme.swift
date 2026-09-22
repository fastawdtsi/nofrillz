import UIKit

enum Theme {
  private enum Palette {
    static let lightBackground = UIColor(red: 245 / 255, green: 243 / 255, blue: 238 / 255, alpha: 1)
    static let lightText = UIColor(red: 17 / 255, green: 17 / 255, blue: 17 / 255, alpha: 1)
    static let lightSecondaryText = UIColor(red: 102 / 255, green: 102 / 255, blue: 102 / 255, alpha: 1)

    static let darkBackground = UIColor(red: 22 / 255, green: 22 / 255, blue: 22 / 255, alpha: 1)
    static let darkText = UIColor(red: 245 / 255, green: 243 / 255, blue: 238 / 255, alpha: 1)
    static let darkSecondaryText = UIColor(red: 168 / 255, green: 163 / 255, blue: 154 / 255, alpha: 1)

    static let lightSurface = UIColor(red: 252 / 255, green: 250 / 255, blue: 246 / 255, alpha: 1)
    static let lightElevatedSurface = UIColor(red: 254 / 255, green: 253 / 255, blue: 250 / 255, alpha: 1)

    static let darkSurface = UIColor(red: 29 / 255, green: 29 / 255, blue: 29 / 255, alpha: 1)
    static let darkElevatedSurface = UIColor(red: 38 / 255, green: 38 / 255, blue: 38 / 255, alpha: 1)
  }

  private enum FontFamily {
    static func uiTextFont(ofSize size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
      contentFont(ofSize: size, weight: weight)
    }

    static func uiDisplayFont(ofSize size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
      contentFont(ofSize: size, weight: weight)
    }

    static func contentFont(ofSize size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
      let fontNames: [String]
      switch weight {
      case .bold, .heavy, .black:
        fontNames = ["IowanOldStyle-Bold", "NewYork-Bold"]
      case .semibold, .medium:
        fontNames = ["IowanOldStyle-Bold", "NewYork-Semibold", "NewYork-Medium"]
      default:
        fontNames = ["IowanOldStyle-Roman", "NewYork-Regular"]
      }
      return namedFont(fontNames, size: size) ?? UIFont.systemFont(ofSize: size, weight: weight)
    }

    private static func namedFont(_ names: [String], size: CGFloat) -> UIFont? {
      for name in names {
        if let font = UIFont(name: name, size: size) {
          return font
        }
      }
      return nil
    }
  }

  private static func dynamicColor(light: UIColor, dark: UIColor) -> UIColor {
    UIColor { traitCollection in
      traitCollection.userInterfaceStyle == .dark ? dark : light
    }
  }

  static let background = dynamicColor(light: Palette.lightBackground, dark: Palette.darkBackground)
  static let text = dynamicColor(light: Palette.lightText, dark: Palette.darkText)
  static let secondaryText = dynamicColor(light: Palette.lightSecondaryText, dark: Palette.darkSecondaryText)
  static let disabledActionText = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? Palette.darkText : Palette.lightText
    return baseColor.withAlphaComponent(0.45)
  }
  static let placeholderText = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? Palette.darkSecondaryText : Palette.lightSecondaryText
    return baseColor.withAlphaComponent(0.8)
  }
  static let primaryMetadataText = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? Palette.darkText : Palette.lightText
    let alpha: CGFloat = traitCollection.userInterfaceStyle == .dark ? 0.88 : 0.8
    return baseColor.withAlphaComponent(alpha)
  }
  static let blue = UIColor(red: 76 / 255, green: 111 / 255, blue: 255 / 255, alpha: 1)
  static let red = UIColor(red: 198 / 255, green: 90 / 255, blue: 90 / 255, alpha: 1)
  static let green = UIColor(red: 93 / 255, green: 138 / 255, blue: 114 / 255, alpha: 1)
  static let yellow = UIColor(red: 201 / 255, green: 164 / 255, blue: 76 / 255, alpha: 1)
  static let orange = UIColor(red: 194 / 255, green: 122 / 255, blue: 61 / 255, alpha: 1)
  static let purple = UIColor(red: 123 / 255, green: 111 / 255, blue: 166 / 255, alpha: 1)
  static let teal = UIColor(red: 79 / 255, green: 140 / 255, blue: 141 / 255, alpha: 1)
  static let likedAccent = red
  static let bookmarkedAccent = yellow
  static let filledControlForeground = dynamicColor(light: Palette.lightBackground, dark: Palette.darkBackground)

  static let separator = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? .white : .black
    let alpha: CGFloat = traitCollection.userInterfaceStyle == .dark ? 0.14 : 0.05
    return baseColor.withAlphaComponent(alpha)
  }

  static let surfaceBackground = UIColor { traitCollection in
    if traitCollection.userInterfaceStyle == .dark {
      return Palette.darkSurface
    }
    return Palette.lightSurface
  }

  static let elevatedSurfaceBackground = UIColor { traitCollection in
    if traitCollection.userInterfaceStyle == .dark {
      return Palette.darkElevatedSurface
    }
    return Palette.lightElevatedSurface
  }

  static let surfaceBorder = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? .white : .black
    let alpha: CGFloat = traitCollection.userInterfaceStyle == .dark ? 0.16 : 0.12
    return baseColor.withAlphaComponent(alpha)
  }

  static let filledControlBackground = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? .white : .black
    return baseColor
  }

  static let chromeShadow = UIColor { traitCollection in
    let baseColor: UIColor = traitCollection.userInterfaceStyle == .dark ? .white : .black
    let alpha: CGFloat = 0.05
    return baseColor.withAlphaComponent(alpha)
  }

  static let postActionIconSize: CGFloat = 28
  static let postActionSpacing: CGFloat = 20
  static let postContentLineSpacing: CGFloat = 0

  static func uiFont(ofSize size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
    FontFamily.uiTextFont(ofSize: size, weight: weight)
  }

  static func uiDisplayFont(ofSize size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
    FontFamily.uiDisplayFont(ofSize: size, weight: weight)
  }

  static func contentFont(ofSize size: CGFloat, weight: UIFont.Weight = .regular) -> UIFont {
    FontFamily.contentFont(ofSize: size, weight: weight)
  }

  enum Typography {
    static var screenTitle: UIFont {
      displayFont(ofSize: 28, weight: .semibold, textStyle: .title2)
    }

    static var navigationTitle: UIFont {
      displayFont(ofSize: 17, weight: .semibold, textStyle: .headline)
    }

    static var navigationSubtitle: UIFont {
      interfaceFont(ofSize: 12, weight: .regular, textStyle: .caption1)
    }

    static var navigationAction: UIFont {
      interfaceFont(ofSize: 17, weight: .medium, textStyle: .body)
    }

    static var primaryAction: UIFont {
      interfaceFont(ofSize: 16, weight: .medium, textStyle: .callout)
    }

    static var primaryButton: UIFont {
      primaryAction
    }

    static var secondaryAction: UIFont {
      interfaceFont(ofSize: 14, weight: .regular, textStyle: .subheadline)
    }

    static var secondaryButton: UIFont {
      secondaryAction
    }

    static var rowTitle: UIFont {
      interfaceFont(ofSize: 17, weight: .semibold, textStyle: .body)
    }

    static var sectionHeading: UIFont {
      rowTitle
    }

    static var inlineTitle: UIFont {
      interfaceFont(ofSize: 15, weight: .semibold, textStyle: .subheadline)
    }

    static var rowSubtitle: UIFont {
      interfaceFont(ofSize: 13, weight: .regular, textStyle: .footnote)
    }

    static var settingsDescription: UIFont {
      rowSubtitle
    }

    static var fieldLabel: UIFont {
      interfaceFont(ofSize: 12, weight: .regular, textStyle: .caption1)
    }

    static var fieldValue: UIFont {
      interfaceFont(ofSize: 17, weight: .regular, textStyle: .body)
    }

    static var username: UIFont {
      interfaceFont(ofSize: 14, weight: .regular, textStyle: .subheadline)
    }

    static var helperText: UIFont {
      interfaceFont(ofSize: 16, weight: .regular, textStyle: .body)
    }

    static var validationMessage: UIFont {
      helperText
    }

    static var metadata: UIFont {
      interfaceFont(ofSize: 13, weight: .regular, textStyle: .footnote)
    }

    static var timestamp: UIFont {
      metadata
    }

    static var caption: UIFont {
      interfaceFont(ofSize: 14, weight: .regular, textStyle: .subheadline)
    }

    static var placeholder: UIFont {
      interfaceFont(ofSize: 17, weight: .regular, textStyle: .body)
    }

    static var characterCounter: UIFont {
      interfaceFont(ofSize: 11, weight: .regular, textStyle: .caption2)
    }

    static var tabLabel: UIFont {
      interfaceFont(ofSize: 10, weight: .regular, textStyle: .caption2)
    }

    static var countPill: UIFont {
      interfaceFont(ofSize: 13, weight: .semibold, textStyle: .footnote)
    }

    static var bodyEditorialSmall: UIFont {
      editorialFont(ofSize: 16, weight: .regular, textStyle: .body)
    }

    static var bio: UIFont {
      bodyEditorialSmall
    }

    static var bodyEditorial: UIFont {
      editorialFont(ofSize: 17, weight: .regular, textStyle: .body)
    }

    static var postBody: UIFont {
      bodyEditorial
    }

    static var composeBody: UIFont {
      bodyEditorialLarge
    }

    static var bodyEditorialMedium: UIFont {
      editorialFont(ofSize: 18, weight: .regular, textStyle: .title3)
    }

    static var editorialExcerpt: UIFont {
      bodyEditorialMedium
    }

    static var bodyEditorialLarge: UIFont {
      editorialFont(ofSize: 20, weight: .regular, textStyle: .title3)
    }

    static var editorialHeadline: UIFont {
      editorialFont(ofSize: 17, weight: .semibold, textStyle: .body)
    }

    static var displayName: UIFont {
      editorialFont(ofSize: 38, weight: .semibold, textStyle: .largeTitle)
    }

    static var profileDisplayName: UIFont {
      editorialFont(ofSize: 30, weight: .semibold, textStyle: .title1)
    }

    static var longFormHeadline: UIFont {
      profileDisplayName
    }

    static var settingsRowTitle: UIFont {
      rowTitle
    }

    static var navigationActionTextAttributes: [NSAttributedString.Key: Any] {
      [
        .font: navigationAction,
        .foregroundColor: Theme.text
      ]
    }

    static var disabledNavigationActionTextAttributes: [NSAttributedString.Key: Any] {
      [
        .font: navigationAction,
        .foregroundColor: Theme.disabledActionText
      ]
    }

    static func placeholderAttributes(color: UIColor = Theme.placeholderText) -> [NSAttributedString.Key: Any] {
      [
        .font: placeholder,
        .foregroundColor: color
      ]
    }

    private static func interfaceFont(
      ofSize size: CGFloat,
      weight: UIFont.Weight,
      textStyle: UIFont.TextStyle
    ) -> UIFont {
      scaledFont(baseFont: FontFamily.uiTextFont(ofSize: size, weight: weight), textStyle: textStyle)
    }

    private static func displayFont(
      ofSize size: CGFloat,
      weight: UIFont.Weight,
      textStyle: UIFont.TextStyle
    ) -> UIFont {
      scaledFont(baseFont: FontFamily.uiDisplayFont(ofSize: size, weight: weight), textStyle: textStyle)
    }

    private static func editorialFont(
      ofSize size: CGFloat,
      weight: UIFont.Weight,
      textStyle: UIFont.TextStyle
    ) -> UIFont {
      scaledFont(baseFont: FontFamily.contentFont(ofSize: size, weight: weight), textStyle: textStyle)
    }

    private static func scaledFont(baseFont: UIFont, textStyle: UIFont.TextStyle) -> UIFont {
      UIFontMetrics(forTextStyle: textStyle).scaledFont(for: baseFont)
    }
  }

  static func applyGlobalAppearance() {
    let navigationBarAppearance = UINavigationBarAppearance()
    configureNavigationBarAppearance(navigationBarAppearance)
    UINavigationBar.appearance().standardAppearance = navigationBarAppearance
    UINavigationBar.appearance().scrollEdgeAppearance = navigationBarAppearance
    UINavigationBar.appearance().compactAppearance = navigationBarAppearance
    UINavigationBar.appearance().tintColor = text
    UINavigationBar.appearance().titleTextAttributes = [.foregroundColor: UIColor.clear]
    if let backImage = AppIcon.chevronLeft.image {
      UINavigationBar.appearance().backIndicatorImage = backImage
      UINavigationBar.appearance().backIndicatorTransitionMaskImage = backImage
    }
    UIBarButtonItem.appearance().tintColor = text
    let hiddenBackTitleOffset = UIOffset(horizontal: -1000, vertical: 0)
    UIBarButtonItem.appearance().setBackButtonTitlePositionAdjustment(hiddenBackTitleOffset, for: .default)
    UIBarButtonItem.appearance().setBackButtonTitlePositionAdjustment(hiddenBackTitleOffset, for: .compact)

    let tabBarAppearance = UITabBarAppearance()
    configureTabBarAppearance(tabBarAppearance)
    UITabBar.appearance().standardAppearance = tabBarAppearance
    UITabBar.appearance().scrollEdgeAppearance = tabBarAppearance
    UITabBar.appearance().tintColor = text
    UITabBar.appearance().unselectedItemTintColor = secondaryText
    UITabBar.appearance().isTranslucent = true

    UITableView.appearance().backgroundColor = background
    UITableViewCell.appearance().backgroundColor = background
  }

  static func configureNavigationBarAppearance(_ appearance: UINavigationBarAppearance) {
    appearance.configureWithTransparentBackground()
    appearance.backgroundColor = .clear
    appearance.backgroundEffect = nil
    appearance.shadowColor = .clear
    appearance.titleTextAttributes = [.foregroundColor: UIColor.clear]
    appearance.largeTitleTextAttributes = [.foregroundColor: UIColor.clear]
    if let backImage = AppIcon.chevronLeft.image {
      appearance.setBackIndicatorImage(backImage, transitionMaskImage: backImage)
    }

    let backButtonAppearance = UIBarButtonItemAppearance()
    backButtonAppearance.normal.titleTextAttributes = [.foregroundColor: UIColor.clear]
    backButtonAppearance.highlighted.titleTextAttributes = [.foregroundColor: UIColor.clear]
    backButtonAppearance.disabled.titleTextAttributes = [.foregroundColor: UIColor.clear]
    appearance.backButtonAppearance = backButtonAppearance
  }

  static func configureTabBarAppearance(_ appearance: UITabBarAppearance) {
    appearance.configureWithOpaqueBackground()
    appearance.backgroundColor = background
    appearance.shadowColor = separator

    appearance.stackedLayoutAppearance.normal.titleTextAttributes = [
      .foregroundColor: secondaryText,
      .font: Typography.tabLabel
    ]
    appearance.stackedLayoutAppearance.selected.titleTextAttributes = [
      .foregroundColor: text,
      .font: Typography.tabLabel
    ]
  }

  static func applyEditorialBarButtonStyle(to item: UIBarButtonItem) {
    if #available(iOS 26.0, *) {
      item.hidesSharedBackground = true
      item.sharesBackground = false
    }
  }

  static func applySurface(to view: UIView, cornerRadius: CGFloat, fillColor: UIColor? = nil) {
    view.layer.cornerRadius = cornerRadius
    view.layer.masksToBounds = true
    view.layer.borderWidth = 1
    view.layer.borderColor = surfaceBorder.cgColor
    view.backgroundColor = fillColor ?? background
  }

  static func applyTextInputStyle(to view: UIView, cornerRadius: CGFloat = 12) {
    applySurface(to: view, cornerRadius: cornerRadius, fillColor: elevatedSurfaceBackground)
  }

  static func applyPillStyle(to view: UIView) {
    applySurface(to: view, cornerRadius: 14, fillColor: elevatedSurfaceBackground)
  }

  static func applyFilledControlStyle(to view: UIView, cornerRadius: CGFloat) {
    view.backgroundColor = filledControlBackground
    view.layer.cornerRadius = cornerRadius
    view.layer.masksToBounds = true
    view.layer.borderWidth = 0
  }
}
