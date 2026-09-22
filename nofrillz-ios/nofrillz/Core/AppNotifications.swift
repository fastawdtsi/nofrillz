import Foundation

struct FollowRelationshipChange {
  let userID: String
  let isFollowing: Bool
}

struct BookmarkStateChange {
  let post: Post
  let isInFlight: Bool
}

struct ToastRequest {
  let message: ToastMessage
}

extension Notification.Name {
  static let postCreated = Notification.Name("postCreated")
  static let followRelationshipDidChange = Notification.Name("followRelationshipDidChange")
  static let bookmarkStateDidChange = Notification.Name("bookmarkStateDidChange")
  static let connectivityStatusDidChange = Notification.Name("connectivityStatusDidChange")
  static let toastMessageRequested = Notification.Name("toastMessageRequested")
}

extension Notification {
  var followRelationshipChange: FollowRelationshipChange? {
    object as? FollowRelationshipChange
  }

  var bookmarkStateChange: BookmarkStateChange? {
    object as? BookmarkStateChange
  }

  var connectivityStatusChange: ConnectivityStatusChange? {
    object as? ConnectivityStatusChange
  }

  var toastMessage: ToastMessage? {
    object as? ToastMessage
  }
}
