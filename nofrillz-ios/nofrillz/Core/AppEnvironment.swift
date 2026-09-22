import Foundation

struct AppEnvironment {
  let api: NoFrillzAPI
  let sessionStore: SessionStore
  let bookmarkStore: BookmarkStore
  let connectivityMonitor: ConnectivityMonitor
  let feedbackCenter: AppFeedbackCenter

  @MainActor
  static func live() -> AppEnvironment {
    let sessionStore = SessionStore()
    let config = APIConfig.fromInfoPlist()
    let api = NoFrillzAPI(config: config, sessionStore: sessionStore)
    let bookmarkStore = BookmarkStore(api: api, sessionStore: sessionStore)
    let connectivityMonitor = ConnectivityMonitor()
    connectivityMonitor.start()
    let feedbackCenter = AppFeedbackCenter(connectivityMonitor: connectivityMonitor)
    return AppEnvironment(
      api: api,
      sessionStore: sessionStore,
      bookmarkStore: bookmarkStore,
      connectivityMonitor: connectivityMonitor,
      feedbackCenter: feedbackCenter
    )
  }
}
