import UIKit

enum AppIcon: String {
  case bookmark = "IconBookmark"
  case bookmarkFilled = "IconBookmarkFilled"
  case check = "IconCheck"
  case chevronLeft = "IconChevronLeft"
  case chevronRight = "IconChevronRight"
  case close = "IconClose"
  case comment = "IconComment"
  case discover = "IconDiscover"
  case heart = "IconHeart"
  case heartFilled = "IconHeartFilled"
  case home = "IconHome"
  case minus = "IconMinus"
  case more = "IconMore"
  case pencil = "IconPencil"
  case plus = "IconPlus"
  case profile = "IconProfile"
  case search = "IconSearch"
  case settings = "IconSettings"
  case share = "IconShare"

  var image: UIImage? {
    UIImage(named: rawValue)?.withRenderingMode(.alwaysTemplate)
  }
}
