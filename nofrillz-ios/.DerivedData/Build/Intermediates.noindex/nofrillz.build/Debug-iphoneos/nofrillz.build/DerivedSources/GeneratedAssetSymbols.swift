import Foundation
#if canImport(AppKit)
import AppKit
#endif
#if canImport(UIKit)
import UIKit
#endif
#if canImport(SwiftUI)
import SwiftUI
#endif
#if canImport(DeveloperToolsSupport)
import DeveloperToolsSupport
#endif

#if SWIFT_PACKAGE
private let resourceBundle = Foundation.Bundle.module
#else
private class ResourceBundleClass {}
private let resourceBundle = Foundation.Bundle(for: ResourceBundleClass.self)
#endif

// MARK: - Color Symbols -

@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
extension DeveloperToolsSupport.ColorResource {

    /// The "LaunchBackgroundColor" asset catalog color resource.
    static let launchBackground = DeveloperToolsSupport.ColorResource(name: "LaunchBackgroundColor", bundle: resourceBundle)

    /// The "LaunchPrimaryTextColor" asset catalog color resource.
    static let launchPrimaryText = DeveloperToolsSupport.ColorResource(name: "LaunchPrimaryTextColor", bundle: resourceBundle)

    /// The "LaunchSecondaryTextColor" asset catalog color resource.
    static let launchSecondaryText = DeveloperToolsSupport.ColorResource(name: "LaunchSecondaryTextColor", bundle: resourceBundle)

}

// MARK: - Image Symbols -

@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
extension DeveloperToolsSupport.ImageResource {

    /// The "FeedBanner" asset catalog image resource.
    static let feedBanner = DeveloperToolsSupport.ImageResource(name: "FeedBanner", bundle: resourceBundle)

    /// The "IconCheck" asset catalog image resource.
    static let iconCheck = DeveloperToolsSupport.ImageResource(name: "IconCheck", bundle: resourceBundle)

    /// The "IconChevronLeft" asset catalog image resource.
    static let iconChevronLeft = DeveloperToolsSupport.ImageResource(name: "IconChevronLeft", bundle: resourceBundle)

    /// The "IconChevronRight" asset catalog image resource.
    static let iconChevronRight = DeveloperToolsSupport.ImageResource(name: "IconChevronRight", bundle: resourceBundle)

    /// The "IconClose" asset catalog image resource.
    static let iconClose = DeveloperToolsSupport.ImageResource(name: "IconClose", bundle: resourceBundle)

    /// The "IconComment" asset catalog image resource.
    static let iconComment = DeveloperToolsSupport.ImageResource(name: "IconComment", bundle: resourceBundle)

    /// The "IconDiscover" asset catalog image resource.
    static let iconDiscover = DeveloperToolsSupport.ImageResource(name: "IconDiscover", bundle: resourceBundle)

    /// The "IconHeart" asset catalog image resource.
    static let iconHeart = DeveloperToolsSupport.ImageResource(name: "IconHeart", bundle: resourceBundle)

    /// The "IconHeartFilled" asset catalog image resource.
    static let iconHeartFilled = DeveloperToolsSupport.ImageResource(name: "IconHeartFilled", bundle: resourceBundle)

    /// The "IconHome" asset catalog image resource.
    static let iconHome = DeveloperToolsSupport.ImageResource(name: "IconHome", bundle: resourceBundle)

    /// The "IconMinus" asset catalog image resource.
    static let iconMinus = DeveloperToolsSupport.ImageResource(name: "IconMinus", bundle: resourceBundle)

    /// The "IconMore" asset catalog image resource.
    static let iconMore = DeveloperToolsSupport.ImageResource(name: "IconMore", bundle: resourceBundle)

    /// The "IconPencil" asset catalog image resource.
    static let iconPencil = DeveloperToolsSupport.ImageResource(name: "IconPencil", bundle: resourceBundle)

    /// The "IconPlus" asset catalog image resource.
    static let iconPlus = DeveloperToolsSupport.ImageResource(name: "IconPlus", bundle: resourceBundle)

    /// The "IconProfile" asset catalog image resource.
    static let iconProfile = DeveloperToolsSupport.ImageResource(name: "IconProfile", bundle: resourceBundle)

    /// The "IconSearch" asset catalog image resource.
    static let iconSearch = DeveloperToolsSupport.ImageResource(name: "IconSearch", bundle: resourceBundle)

    /// The "IconSettings" asset catalog image resource.
    static let iconSettings = DeveloperToolsSupport.ImageResource(name: "IconSettings", bundle: resourceBundle)

    /// The "IconShare" asset catalog image resource.
    static let iconShare = DeveloperToolsSupport.ImageResource(name: "IconShare", bundle: resourceBundle)

    /// The "LaunchBackground" asset catalog image resource.
    static let launchBackground = DeveloperToolsSupport.ImageResource(name: "LaunchBackground", bundle: resourceBundle)

}

// MARK: - Color Symbol Extensions -

#if canImport(AppKit)
@available(macOS 14.0, *)
@available(macCatalyst, unavailable)
extension AppKit.NSColor {

    /// The "LaunchBackgroundColor" asset catalog color.
    static var launchBackground: AppKit.NSColor {
#if !targetEnvironment(macCatalyst)
        .init(resource: .launchBackground)
#else
        .init()
#endif
    }

    /// The "LaunchPrimaryTextColor" asset catalog color.
    static var launchPrimaryText: AppKit.NSColor {
#if !targetEnvironment(macCatalyst)
        .init(resource: .launchPrimaryText)
#else
        .init()
#endif
    }

    /// The "LaunchSecondaryTextColor" asset catalog color.
    static var launchSecondaryText: AppKit.NSColor {
#if !targetEnvironment(macCatalyst)
        .init(resource: .launchSecondaryText)
#else
        .init()
#endif
    }

}
#endif

#if canImport(UIKit)
@available(iOS 17.0, tvOS 17.0, *)
@available(watchOS, unavailable)
extension UIKit.UIColor {

    /// The "LaunchBackgroundColor" asset catalog color.
    static var launchBackground: UIKit.UIColor {
#if !os(watchOS)
        .init(resource: .launchBackground)
#else
        .init()
#endif
    }

    /// The "LaunchPrimaryTextColor" asset catalog color.
    static var launchPrimaryText: UIKit.UIColor {
#if !os(watchOS)
        .init(resource: .launchPrimaryText)
#else
        .init()
#endif
    }

    /// The "LaunchSecondaryTextColor" asset catalog color.
    static var launchSecondaryText: UIKit.UIColor {
#if !os(watchOS)
        .init(resource: .launchSecondaryText)
#else
        .init()
#endif
    }

}
#endif

#if canImport(SwiftUI)
@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
extension SwiftUI.Color {

    /// The "LaunchBackgroundColor" asset catalog color.
    static var launchBackground: SwiftUI.Color { .init(.launchBackground) }

    /// The "LaunchPrimaryTextColor" asset catalog color.
    static var launchPrimaryText: SwiftUI.Color { .init(.launchPrimaryText) }

    /// The "LaunchSecondaryTextColor" asset catalog color.
    static var launchSecondaryText: SwiftUI.Color { .init(.launchSecondaryText) }

}

@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
extension SwiftUI.ShapeStyle where Self == SwiftUI.Color {

    /// The "LaunchBackgroundColor" asset catalog color.
    static var launchBackground: SwiftUI.Color { .init(.launchBackground) }

    /// The "LaunchPrimaryTextColor" asset catalog color.
    static var launchPrimaryText: SwiftUI.Color { .init(.launchPrimaryText) }

    /// The "LaunchSecondaryTextColor" asset catalog color.
    static var launchSecondaryText: SwiftUI.Color { .init(.launchSecondaryText) }

}
#endif

// MARK: - Image Symbol Extensions -

#if canImport(AppKit)
@available(macOS 14.0, *)
@available(macCatalyst, unavailable)
extension AppKit.NSImage {

    /// The "FeedBanner" asset catalog image.
    static var feedBanner: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .feedBanner)
#else
        .init()
#endif
    }

    /// The "IconCheck" asset catalog image.
    static var iconCheck: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconCheck)
#else
        .init()
#endif
    }

    /// The "IconChevronLeft" asset catalog image.
    static var iconChevronLeft: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconChevronLeft)
#else
        .init()
#endif
    }

    /// The "IconChevronRight" asset catalog image.
    static var iconChevronRight: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconChevronRight)
#else
        .init()
#endif
    }

    /// The "IconClose" asset catalog image.
    static var iconClose: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconClose)
#else
        .init()
#endif
    }

    /// The "IconComment" asset catalog image.
    static var iconComment: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconComment)
#else
        .init()
#endif
    }

    /// The "IconDiscover" asset catalog image.
    static var iconDiscover: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconDiscover)
#else
        .init()
#endif
    }

    /// The "IconHeart" asset catalog image.
    static var iconHeart: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconHeart)
#else
        .init()
#endif
    }

    /// The "IconHeartFilled" asset catalog image.
    static var iconHeartFilled: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconHeartFilled)
#else
        .init()
#endif
    }

    /// The "IconHome" asset catalog image.
    static var iconHome: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconHome)
#else
        .init()
#endif
    }

    /// The "IconMinus" asset catalog image.
    static var iconMinus: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconMinus)
#else
        .init()
#endif
    }

    /// The "IconMore" asset catalog image.
    static var iconMore: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconMore)
#else
        .init()
#endif
    }

    /// The "IconPencil" asset catalog image.
    static var iconPencil: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconPencil)
#else
        .init()
#endif
    }

    /// The "IconPlus" asset catalog image.
    static var iconPlus: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconPlus)
#else
        .init()
#endif
    }

    /// The "IconProfile" asset catalog image.
    static var iconProfile: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconProfile)
#else
        .init()
#endif
    }

    /// The "IconSearch" asset catalog image.
    static var iconSearch: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconSearch)
#else
        .init()
#endif
    }

    /// The "IconSettings" asset catalog image.
    static var iconSettings: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconSettings)
#else
        .init()
#endif
    }

    /// The "IconShare" asset catalog image.
    static var iconShare: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .iconShare)
#else
        .init()
#endif
    }

    /// The "LaunchBackground" asset catalog image.
    static var launchBackground: AppKit.NSImage {
#if !targetEnvironment(macCatalyst)
        .init(resource: .launchBackground)
#else
        .init()
#endif
    }

}
#endif

#if canImport(UIKit)
@available(iOS 17.0, tvOS 17.0, *)
@available(watchOS, unavailable)
extension UIKit.UIImage {

    /// The "FeedBanner" asset catalog image.
    static var feedBanner: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .feedBanner)
#else
        .init()
#endif
    }

    /// The "IconCheck" asset catalog image.
    static var iconCheck: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconCheck)
#else
        .init()
#endif
    }

    /// The "IconChevronLeft" asset catalog image.
    static var iconChevronLeft: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconChevronLeft)
#else
        .init()
#endif
    }

    /// The "IconChevronRight" asset catalog image.
    static var iconChevronRight: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconChevronRight)
#else
        .init()
#endif
    }

    /// The "IconClose" asset catalog image.
    static var iconClose: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconClose)
#else
        .init()
#endif
    }

    /// The "IconComment" asset catalog image.
    static var iconComment: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconComment)
#else
        .init()
#endif
    }

    /// The "IconDiscover" asset catalog image.
    static var iconDiscover: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconDiscover)
#else
        .init()
#endif
    }

    /// The "IconHeart" asset catalog image.
    static var iconHeart: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconHeart)
#else
        .init()
#endif
    }

    /// The "IconHeartFilled" asset catalog image.
    static var iconHeartFilled: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconHeartFilled)
#else
        .init()
#endif
    }

    /// The "IconHome" asset catalog image.
    static var iconHome: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconHome)
#else
        .init()
#endif
    }

    /// The "IconMinus" asset catalog image.
    static var iconMinus: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconMinus)
#else
        .init()
#endif
    }

    /// The "IconMore" asset catalog image.
    static var iconMore: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconMore)
#else
        .init()
#endif
    }

    /// The "IconPencil" asset catalog image.
    static var iconPencil: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconPencil)
#else
        .init()
#endif
    }

    /// The "IconPlus" asset catalog image.
    static var iconPlus: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconPlus)
#else
        .init()
#endif
    }

    /// The "IconProfile" asset catalog image.
    static var iconProfile: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconProfile)
#else
        .init()
#endif
    }

    /// The "IconSearch" asset catalog image.
    static var iconSearch: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconSearch)
#else
        .init()
#endif
    }

    /// The "IconSettings" asset catalog image.
    static var iconSettings: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconSettings)
#else
        .init()
#endif
    }

    /// The "IconShare" asset catalog image.
    static var iconShare: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .iconShare)
#else
        .init()
#endif
    }

    /// The "LaunchBackground" asset catalog image.
    static var launchBackground: UIKit.UIImage {
#if !os(watchOS)
        .init(resource: .launchBackground)
#else
        .init()
#endif
    }

}
#endif

// MARK: - Thinnable Asset Support -

@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
@available(watchOS, unavailable)
extension DeveloperToolsSupport.ColorResource {

    private init?(thinnableName: Swift.String, bundle: Foundation.Bundle) {
#if canImport(AppKit) && os(macOS)
        if AppKit.NSColor(named: NSColor.Name(thinnableName), bundle: bundle) != nil {
            self.init(name: thinnableName, bundle: bundle)
        } else {
            return nil
        }
#elseif canImport(UIKit) && !os(watchOS)
        if UIKit.UIColor(named: thinnableName, in: bundle, compatibleWith: nil) != nil {
            self.init(name: thinnableName, bundle: bundle)
        } else {
            return nil
        }
#else
        return nil
#endif
    }

}

#if canImport(AppKit)
@available(macOS 14.0, *)
@available(macCatalyst, unavailable)
extension AppKit.NSColor {

    private convenience init?(thinnableResource: DeveloperToolsSupport.ColorResource?) {
#if !targetEnvironment(macCatalyst)
        if let resource = thinnableResource {
            self.init(resource: resource)
        } else {
            return nil
        }
#else
        return nil
#endif
    }

}
#endif

#if canImport(UIKit)
@available(iOS 17.0, tvOS 17.0, *)
@available(watchOS, unavailable)
extension UIKit.UIColor {

    private convenience init?(thinnableResource: DeveloperToolsSupport.ColorResource?) {
#if !os(watchOS)
        if let resource = thinnableResource {
            self.init(resource: resource)
        } else {
            return nil
        }
#else
        return nil
#endif
    }

}
#endif

#if canImport(SwiftUI)
@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
extension SwiftUI.Color {

    private init?(thinnableResource: DeveloperToolsSupport.ColorResource?) {
        if let resource = thinnableResource {
            self.init(resource)
        } else {
            return nil
        }
    }

}

@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
extension SwiftUI.ShapeStyle where Self == SwiftUI.Color {

    private init?(thinnableResource: DeveloperToolsSupport.ColorResource?) {
        if let resource = thinnableResource {
            self.init(resource)
        } else {
            return nil
        }
    }

}
#endif

@available(iOS 17.0, macOS 14.0, tvOS 17.0, watchOS 10.0, *)
@available(watchOS, unavailable)
extension DeveloperToolsSupport.ImageResource {

    private init?(thinnableName: Swift.String, bundle: Foundation.Bundle) {
#if canImport(AppKit) && os(macOS)
        if bundle.image(forResource: NSImage.Name(thinnableName)) != nil {
            self.init(name: thinnableName, bundle: bundle)
        } else {
            return nil
        }
#elseif canImport(UIKit) && !os(watchOS)
        if UIKit.UIImage(named: thinnableName, in: bundle, compatibleWith: nil) != nil {
            self.init(name: thinnableName, bundle: bundle)
        } else {
            return nil
        }
#else
        return nil
#endif
    }

}

#if canImport(AppKit)
@available(macOS 14.0, *)
@available(macCatalyst, unavailable)
extension AppKit.NSImage {

    private convenience init?(thinnableResource: DeveloperToolsSupport.ImageResource?) {
#if !targetEnvironment(macCatalyst)
        if let resource = thinnableResource {
            self.init(resource: resource)
        } else {
            return nil
        }
#else
        return nil
#endif
    }

}
#endif

#if canImport(UIKit)
@available(iOS 17.0, tvOS 17.0, *)
@available(watchOS, unavailable)
extension UIKit.UIImage {

    private convenience init?(thinnableResource: DeveloperToolsSupport.ImageResource?) {
#if !os(watchOS)
        if let resource = thinnableResource {
            self.init(resource: resource)
        } else {
            return nil
        }
#else
        return nil
#endif
    }

}
#endif

