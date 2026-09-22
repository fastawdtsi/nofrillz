import Foundation
import Security

extension Notification.Name {
  static let sessionDidChange = Notification.Name("sessionDidChange")
}

final class SessionStore {
  private enum Keys {
    static let authToken = "nofrillz.authToken"
    static let refreshToken = "nofrillz.refreshToken"
    static let userID = "nofrillz.userID"
    static let username = "nofrillz.username"
    static let firstName = "nofrillz.firstName"
    static let lastName = "nofrillz.lastName"
    static let email = "nofrillz.email"
    static let about = "nofrillz.about"
    static let avatarURL = "nofrillz.avatarURL"
    static let followerCount = "nofrillz.followerCount"
    static let followingCount = "nofrillz.followingCount"
    static let postCount = "nofrillz.postCount"
  }

  private enum KeychainKeys {
    static let authToken = "nofrillz.keychain.authToken"
    static let refreshToken = "nofrillz.keychain.refreshToken"
  }

  private let defaults: UserDefaults
  private let keychainService: String

  var authToken: String? {
    keychainValue(for: KeychainKeys.authToken)
  }

  var refreshToken: String? {
    keychainValue(for: KeychainKeys.refreshToken)
  }

  var currentUser: User? {
    guard let id = defaults.string(forKey: Keys.userID) else { return nil }
    let user = User(
      id: id,
      username: defaults.string(forKey: Keys.username) ?? "",
      firstName: defaults.string(forKey: Keys.firstName),
      lastName: defaults.string(forKey: Keys.lastName),
      email: defaults.string(forKey: Keys.email),
      about: defaults.string(forKey: Keys.about),
      avatarURL: defaults.string(forKey: Keys.avatarURL),
      joinedAt: nil,
      isFollowing: false,
      isFollower: false,
      followerCount: defaults.integer(forKey: Keys.followerCount),
      followingCount: defaults.integer(forKey: Keys.followingCount),
      postCount: defaults.integer(forKey: Keys.postCount)
    )
    debugLogUser("currentUser", user: user)
    return user
  }

  var isAuthenticated: Bool {
    if let token = authToken {
      return !token.isEmpty
    }
    return false
  }

  init(defaults: UserDefaults = .standard) {
    self.defaults = defaults
    keychainService = Bundle.main.bundleIdentifier ?? "com.rossotech.nofrillz"
    migrateLegacyTokensFromUserDefaults()
  }

  private func postSessionDidChange() {
    if Thread.isMainThread {
      NotificationCenter.default.post(name: .sessionDidChange, object: nil)
    } else {
      DispatchQueue.main.async {
        NotificationCenter.default.post(name: .sessionDidChange, object: nil)
      }
    }
  }

  func saveSession(token: String, refreshToken: String?, user: User?, userID: String?) {
    setKeychainValue(token, for: KeychainKeys.authToken)
    if let refreshToken, !refreshToken.isEmpty {
      setKeychainValue(refreshToken, for: KeychainKeys.refreshToken)
    } else {
      removeKeychainValue(for: KeychainKeys.refreshToken)
    }

    if let user {
      debugLogUser("saveSession(input)", user: user)
      defaults.set(user.id, forKey: Keys.userID)
      defaults.set(user.username, forKey: Keys.username)
      defaults.set(user.firstName, forKey: Keys.firstName)
      defaults.set(user.lastName, forKey: Keys.lastName)
      defaults.set(user.email, forKey: Keys.email)
      defaults.set(user.about, forKey: Keys.about)
      defaults.set(user.avatarURL, forKey: Keys.avatarURL)
      defaults.set(user.followerCount, forKey: Keys.followerCount)
      defaults.set(user.followingCount, forKey: Keys.followingCount)
      defaults.set(user.postCount, forKey: Keys.postCount)
    } else if let userID, !userID.isEmpty {
      defaults.set(userID, forKey: Keys.userID)
    }

    postSessionDidChange()
  }

  func updateUser(_ user: User) {
    debugLogUser("updateUser(input)", user: user)
    defaults.set(user.id, forKey: Keys.userID)
    defaults.set(user.username, forKey: Keys.username)
    defaults.set(user.firstName, forKey: Keys.firstName)
    defaults.set(user.lastName, forKey: Keys.lastName)
    defaults.set(user.email, forKey: Keys.email)
    defaults.set(user.about, forKey: Keys.about)
    defaults.set(user.avatarURL, forKey: Keys.avatarURL)
    defaults.set(user.followerCount, forKey: Keys.followerCount)
    defaults.set(user.followingCount, forKey: Keys.followingCount)
    defaults.set(user.postCount, forKey: Keys.postCount)
  }

  func adjustCurrentUserFollowingCount(by delta: Int) {
    guard let currentUser else { return }
    let updatedFollowingCount = max(0, currentUser.followingCount + delta)
    updateUser(currentUser.withFollowingCount(updatedFollowingCount))
  }

  func clear() {
    removeKeychainValue(for: KeychainKeys.authToken)
    removeKeychainValue(for: KeychainKeys.refreshToken)
    defaults.removeObject(forKey: Keys.authToken)
    defaults.removeObject(forKey: Keys.refreshToken)
    defaults.removeObject(forKey: Keys.userID)
    defaults.removeObject(forKey: Keys.username)
    defaults.removeObject(forKey: Keys.firstName)
    defaults.removeObject(forKey: Keys.lastName)
    defaults.removeObject(forKey: Keys.email)
    defaults.removeObject(forKey: Keys.about)
    defaults.removeObject(forKey: Keys.avatarURL)
    defaults.removeObject(forKey: Keys.followerCount)
    defaults.removeObject(forKey: Keys.followingCount)
    defaults.removeObject(forKey: Keys.postCount)
    postSessionDidChange()
  }

  private func debugLogUser(_ context: String, user: User?) {
#if DEBUG
    guard let user else {
      print("[NoFrillzDebug][SessionStore][\(context)] user=nil")
      return
    }
    print(
      "[NoFrillzDebug][SessionStore][\(context)] id=\(user.id) username=\(user.username) first=\(user.firstName ?? "nil") last=\(user.lastName ?? "nil") postCount=\(user.postCount)"
    )
#endif
  }

  private func migrateLegacyTokensFromUserDefaults() {
    if keychainValue(for: KeychainKeys.authToken) == nil,
       let legacyAuthToken = defaults.string(forKey: Keys.authToken),
       !legacyAuthToken.isEmpty {
      setKeychainValue(legacyAuthToken, for: KeychainKeys.authToken)
    }

    if keychainValue(for: KeychainKeys.refreshToken) == nil,
       let legacyRefreshToken = defaults.string(forKey: Keys.refreshToken),
       !legacyRefreshToken.isEmpty {
      setKeychainValue(legacyRefreshToken, for: KeychainKeys.refreshToken)
    }

    defaults.removeObject(forKey: Keys.authToken)
    defaults.removeObject(forKey: Keys.refreshToken)
  }

  private func keychainValue(for account: String) -> String? {
    let query: [String: Any] = [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: keychainService,
      kSecAttrAccount as String: account,
      kSecReturnData as String: true,
      kSecMatchLimit as String: kSecMatchLimitOne
    ]

    var result: AnyObject?
    let status = SecItemCopyMatching(query as CFDictionary, &result)
    guard status == errSecSuccess else { return nil }
    guard let data = result as? Data else { return nil }
    return String(data: data, encoding: .utf8)
  }

  private func setKeychainValue(_ value: String, for account: String) {
    guard let data = value.data(using: .utf8) else { return }

    let query: [String: Any] = [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: keychainService,
      kSecAttrAccount as String: account
    ]

    let attributes: [String: Any] = [
      kSecValueData as String: data
    ]

    let status = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
    if status == errSecSuccess { return }

    var item = query
    item[kSecValueData as String] = data
    SecItemAdd(item as CFDictionary, nil)
  }

  private func removeKeychainValue(for account: String) {
    let query: [String: Any] = [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: keychainService,
      kSecAttrAccount as String: account
    ]
    SecItemDelete(query as CFDictionary)
  }
}
