import Foundation

struct ToastMessage: Equatable {
  let id = UUID()
  let message: String
  let duration: TimeInterval
  let deduplicationKey: String?
  let accessibilityAnnouncement: String?

  init(
    message: String,
    duration: TimeInterval = 3,
    deduplicationKey: String? = nil,
    accessibilityAnnouncement: String? = nil
  ) {
    self.message = message
    self.duration = duration
    self.deduplicationKey = deduplicationKey
    self.accessibilityAnnouncement = accessibilityAnnouncement ?? message
  }
}

enum RecoverableActionError {
  case respect
  case removeRespect
  case saveBookmark
  case removeBookmark
  case follow
  case unfollow
  case publishPost
  case changePassword
  case signIn
  case signUp
  case saveProfile
  case refreshFeed
  case refreshBookmarks
  case loadSearchResults
  case loadConnections
  case refreshProfile

  var message: String {
    switch self {
    case .respect:
      "Couldn't Respect this post."
    case .removeRespect:
      "Couldn't remove Respect."
    case .saveBookmark:
      "Couldn't save to Bookmarks."
    case .removeBookmark:
      "Couldn't remove from Bookmarks."
    case .follow:
      "Couldn't follow this person."
    case .unfollow:
      "Couldn't unfollow this person."
    case .publishPost:
      "Couldn't publish this post."
    case .changePassword:
      "Couldn't update your password."
    case .signIn:
      "Couldn't sign in."
    case .signUp:
      "Couldn't create your account."
    case .saveProfile:
      "Couldn't save your profile."
    case .refreshFeed:
      "Couldn't refresh the feed."
    case .refreshBookmarks:
      "Couldn't refresh your Bookmarks."
    case .loadSearchResults:
      "Couldn't load search results."
    case .loadConnections:
      "Couldn't load this list."
    case .refreshProfile:
      "Couldn't refresh this profile."
    }
  }

  var deduplicationKey: String {
    switch self {
    case .respect:
      "recoverable-action-respect"
    case .removeRespect:
      "recoverable-action-remove-respect"
    case .saveBookmark:
      "recoverable-action-save-bookmark"
    case .removeBookmark:
      "recoverable-action-remove-bookmark"
    case .follow:
      "recoverable-action-follow"
    case .unfollow:
      "recoverable-action-unfollow"
    case .publishPost:
      "recoverable-action-publish-post"
    case .changePassword:
      "recoverable-action-change-password"
    case .signIn:
      "recoverable-action-sign-in"
    case .signUp:
      "recoverable-action-sign-up"
    case .saveProfile:
      "recoverable-action-save-profile"
    case .refreshFeed:
      "recoverable-action-refresh-feed"
    case .refreshBookmarks:
      "recoverable-action-refresh-bookmarks"
    case .loadSearchResults:
      "recoverable-action-load-search-results"
    case .loadConnections:
      "recoverable-action-load-connections"
    case .refreshProfile:
      "recoverable-action-refresh-profile"
    }
  }
}

@MainActor
final class AppFeedbackCenter {
  private let connectivityMonitor: ConnectivityMonitor
  private var lastPresentedAtByKey: [String: Date] = [:]

  init(connectivityMonitor: ConnectivityMonitor) {
    self.connectivityMonitor = connectivityMonitor
  }

  func showToast(_ message: ToastMessage) {
    if let key = message.deduplicationKey {
      let now = Date()
      if let lastPresentedAt = lastPresentedAtByKey[key], now.timeIntervalSince(lastPresentedAt) < 1.5 {
        return
      }
      lastPresentedAtByKey[key] = now
    }

    NotificationCenter.default.post(name: .toastMessageRequested, object: message)
  }

  func showRecoverableActionFailure(_ error: RecoverableActionError) {
    showToast(
      ToastMessage(
        message: error.message,
        deduplicationKey: error.deduplicationKey
      )
    )
  }

  func showOfflineToastIfNeeded() {
    guard connectivityMonitor.isOffline else { return }
    showToast(
      ToastMessage(
        message: "You're offline.",
        deduplicationKey: "connectivity-offline-toast"
      )
    )
  }
}
