import Foundation

enum APIError: LocalizedError {
  case invalidURL
  case invalidResponse
  case unauthorized
  case server(String)
  case decoding

  var errorDescription: String? {
    switch self {
    case .invalidURL:
      return L10n.tr("api.error.invalid_url")
    case .invalidResponse:
      return L10n.tr("api.error.invalid_response")
    case .unauthorized:
      return L10n.tr("api.error.unauthorized")
    case .server(let message):
      return message
    case .decoding:
      return L10n.tr("api.error.decoding")
    }
  }
}

struct CursorPage<Item> {
  let items: [Item]
  let nextCursor: String?
}

final class NoFrillzAPI {
  private let config: APIConfig
  private let sessionStore: SessionStore
  private let urlSession: URLSession

  init(config: APIConfig, sessionStore: SessionStore, urlSession: URLSession = .shared) {
    self.config = config
    self.sessionStore = sessionStore
    self.urlSession = urlSession
  }

  func resolveURL(path: String?) -> URL? {
    guard let path else { return nil }
    let trimmed = path.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else { return nil }

    if let absoluteURL = URL(string: trimmed), absoluteURL.scheme != nil {
      return absoluteURL
    }

    return URL(string: trimmed, relativeTo: config.baseURL)?.absoluteURL
  }

  func login(email: String, password: String) async throws -> AuthResult {
    let payload = ["email": email, "password": password]
    let data = try await request(path: "/sessions", method: "POST", body: payload, needsAuth: false)
    let result = try decodeAuthResult(from: data)
    debugLogAuthResponse(context: "login", result: result, payload: data)
    return result
  }

  func signup(
    firstName: String,
    lastName: String,
    username: String,
    email: String,
    password: String,
    about: String?
  ) async throws -> AuthResult {
    let payload = SignupRequest(
      firstName: firstName,
      lastName: lastName,
      username: username,
      email: email,
      password: password,
      about: about
    )
    _ = try await request(path: "/users", method: "POST", body: payload, needsAuth: false)
    return try await login(email: email, password: password)
  }

  func fetchFeed() async throws -> [Post] {
    try await fetchFeedPage().items
  }

  func fetchFeedPage(limit: Int = 25, cursor: String? = nil) async throws -> CursorPage<Post> {
    var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
    if let cursor, !cursor.isEmpty {
      queryItems.append(URLQueryItem(name: "cursor", value: cursor))
    }
    let data = try await request(path: "/feed", method: "GET", queryItems: queryItems, needsAuth: true)
    let page = try decodePostsPage(from: data, listKeys: ["posts"])
    debugLogFeedResponse(page: page, payload: data, cursor: cursor)
    return page
  }

  func fetchPost(id: String) async throws -> Post {
    let data = try await request(path: "/posts/\(id)", method: "GET", needsAuth: true)
    return try decodeObject(Post.self, from: data)
  }

  func createPost(content: String) async throws -> Post {
    let payload = CreatePostRequest(body: content)
    let data = try await request(path: "/posts", method: "POST", body: payload, needsAuth: true)
    return try decodeObject(Post.self, from: data)
  }

  func likePost(id: String) async throws {
    _ = try await request(path: "/posts/\(id)/like", method: "POST", needsAuth: true)
  }

  func unlikePost(id: String) async throws {
    _ = try await request(path: "/posts/\(id)/like", method: "DELETE", needsAuth: true)
  }

  func bookmarkPost(id: String) async throws {
    _ = try await request(path: "/posts/\(id)/bookmark", method: "POST", needsAuth: true)
  }

  func removeBookmark(id: String) async throws {
    _ = try await request(path: "/posts/\(id)/bookmark", method: "DELETE", needsAuth: true)
  }

  func fetchBookmarksPage(limit: Int = 20, cursor: String? = nil) async throws -> CursorPage<Post> {
    var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
    if let cursor, !cursor.isEmpty {
      queryItems.append(URLQueryItem(name: "cursor", value: cursor))
    }
    let data = try await request(path: "/bookmarks", method: "GET", queryItems: queryItems, needsAuth: true)
    return try decodeBookmarksPage(from: data)
  }

  func fetchCurrentUser() async throws -> User {
    guard let userID = sessionStore.currentUser?.id, !userID.isEmpty else {
      throw APIError.server(L10n.tr("api.error.missing_current_user_id"))
    }
    let data = try await request(path: "/users/\(userID)", method: "GET", needsAuth: true)
    let user = try decodeObject(User.self, from: data)
    debugLogUserResponse(context: "fetchCurrentUser", user: user, payload: data)
    return user
  }

  func fetchUser(id: String) async throws -> User {
    let data = try await request(path: "/users/\(id)", method: "GET", needsAuth: true)
    return try decodeObject(User.self, from: data)
  }

  func fetchPosts(for userID: String) async throws -> [Post] {
    try await fetchPostsPage(for: userID).items
  }

  func fetchPostsPage(for userID: String, limit: Int = 20, cursor: String? = nil) async throws -> CursorPage<Post> {
    var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
    if let cursor, !cursor.isEmpty {
      queryItems.append(URLQueryItem(name: "before_id", value: cursor))
    }
    let data = try await request(
      path: "/users/\(userID)/posts",
      method: "GET",
      queryItems: queryItems,
      needsAuth: true
    )
    return try decodePostsPage(from: data, listKeys: ["posts"])
  }

  func fetchTopics() async throws -> [Topic] {
    let data = try await request(path: "/topics", method: "GET", needsAuth: false)
    return try decodeTopicsList(from: data)
  }

  func searchUsers(query: String, limit: Int = 20) async throws -> [User] {
    let data = try await request(
      path: "/users/search",
      method: "GET",
      queryItems: [
        URLQueryItem(name: "q", value: query),
        URLQueryItem(name: "limit", value: String(limit))
      ],
      needsAuth: true
    )
    return try decodeUsersList(from: data)
  }

  func followUser(userID: String) async throws {
    _ = try await request(path: "/users/\(userID)/follow", method: "POST", needsAuth: true)
  }

  func unfollowUser(userID: String) async throws {
    _ = try await request(path: "/users/\(userID)/unfollow", method: "POST", needsAuth: true)
  }

  func registerPushDevice(token: String) async throws {
    let payload = RegisterPushDeviceRequest(token: token)
    _ = try await request(path: "/users/apns/device-tokens", method: "PUT", body: payload, needsAuth: true)
  }

  func fetchFollowers(for userID: String) async throws -> [User] {
    try await fetchFollowersPage(for: userID).items
  }

  func fetchFollowersPage(for userID: String, limit: Int = 25, cursor: String? = nil) async throws -> CursorPage<User> {
    var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
    if let cursor, !cursor.isEmpty {
      queryItems.append(URLQueryItem(name: "cursor", value: cursor))
    }
    let data = try await request(
      path: "/users/\(userID)/followers",
      method: "GET",
      queryItems: queryItems,
      needsAuth: true
    )
    return try decodeUsersPage(from: data, keys: ["followers", "users"])
  }

  func fetchFollowing(for userID: String) async throws -> [User] {
    try await fetchFollowingPage(for: userID).items
  }

  func fetchFollowingPage(for userID: String, limit: Int = 25, cursor: String? = nil) async throws -> CursorPage<User> {
    var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
    if let cursor, !cursor.isEmpty {
      queryItems.append(URLQueryItem(name: "cursor", value: cursor))
    }
    let data = try await request(
      path: "/users/\(userID)/following",
      method: "GET",
      queryItems: queryItems,
      needsAuth: true
    )
    if let page = try? decodeUsersPage(from: data, keys: ["following", "users"]) {
      return page
    }

    let ids = try decodeIDList(from: data)
    var users: [User] = []
    users.reserveCapacity(ids.count)
    for id in ids {
      let user = try await fetchUser(id: id)
      users.append(user)
    }
    let nextCursor = ids.count >= limit ? ids.last : nil
    return CursorPage(items: users, nextCursor: nextCursor)
  }

  func signOutCurrentSession() async throws {
    _ = try await request(path: "/sessions/current", method: "DELETE", needsAuth: true)
  }

  func updateCurrentUser(
    firstName: String,
    lastName: String,
    email: String,
    about: String?
  ) async throws -> User {
    guard let userID = sessionStore.currentUser?.id, !userID.isEmpty else {
      throw APIError.server(L10n.tr("api.error.missing_current_user_id"))
    }

    let payload = UpdateCurrentUserRequest(
      firstName: firstName,
      lastName: lastName,
      email: email,
      about: about
    )
    let data = try await request(path: "/users/\(userID)", method: "PATCH", body: payload, needsAuth: true)
    let user = try decodeObject(User.self, from: data)
    debugLogUserResponse(context: "updateCurrentUser", user: user, payload: data)
    return user
  }

  func changeCurrentUserPassword(currentPassword: String, newPassword: String) async throws {
    guard let userID = sessionStore.currentUser?.id, !userID.isEmpty else {
      throw APIError.server(L10n.tr("api.error.missing_current_user_id"))
    }

    let payload = ChangePasswordRequest(
      currentPassword: currentPassword,
      newPassword: newPassword
    )
    _ = try await request(path: "/users/\(userID)/password", method: "PATCH", body: payload, needsAuth: true)
  }

  private func refreshSession() async throws {
    guard let refreshToken = sessionStore.refreshToken, !refreshToken.isEmpty else {
      throw APIError.unauthorized
    }

    let payload = ["refresh_token": refreshToken]
    let data = try await request(
      path: "/sessions/refresh",
      method: "POST",
      body: payload,
      needsAuth: false,
      retryOnUnauthorized: false
    )

    let refreshed = try decodeAuthResult(from: data)
    debugLogAuthResponse(context: "refreshSession", result: refreshed, payload: data)
    let persistedRefreshToken = refreshed.refreshToken ?? refreshToken
    sessionStore.saveSession(
      token: refreshed.token,
      refreshToken: persistedRefreshToken,
      user: refreshed.user,
      userID: refreshed.userID ?? sessionStore.currentUser?.id
    )
  }

  private func request(
    path: String,
    method: String,
    body: (any Encodable)? = nil,
    queryItems: [URLQueryItem] = [],
    needsAuth: Bool,
    retryOnUnauthorized: Bool = true
  ) async throws -> Data {
    guard let baseURL = URL(string: path, relativeTo: config.baseURL),
          var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: true) else {
      throw APIError.invalidURL
    }
    if !queryItems.isEmpty {
      components.queryItems = queryItems
    }
    guard let url = components.url else {
      throw APIError.invalidURL
    }

    var urlRequest = URLRequest(url: url)
    urlRequest.httpMethod = method
    urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")

    if needsAuth, let token = sessionStore.authToken, !token.isEmpty {
      urlRequest.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    }

    if let body {
      urlRequest.httpBody = try JSONEncoder().encode(AnyEncodable(body))
    }

    let (data, response) = try await urlSession.data(for: urlRequest)

    guard let httpResponse = response as? HTTPURLResponse else {
      throw APIError.invalidResponse
    }

    guard (200 ... 299).contains(httpResponse.statusCode) else {
      if httpResponse.statusCode == 401 {
        if needsAuth, retryOnUnauthorized {
          do {
            try await refreshSession()
            return try await request(
              path: path,
              method: method,
              body: body,
              queryItems: queryItems,
              needsAuth: needsAuth,
              retryOnUnauthorized: false
            )
          } catch {
            sessionStore.clear()
            throw APIError.unauthorized
          }
        }

        if needsAuth {
          sessionStore.clear()
        }
        throw APIError.unauthorized
      }

      if let message = try? extractMessage(from: data) {
        throw APIError.server(message)
      }

      throw APIError.server(L10n.tr("api.error.request_failed_status", httpResponse.statusCode))
    }

    return data
  }

  private func decodeAuthResult(from data: Data) throws -> AuthResult {
    if let authResponse = try? JSONDecoder().decode(AuthResponse.self, from: data) {
      if !authResponse.token.isEmpty {
        return AuthResult(
          token: authResponse.token,
          refreshToken: authResponse.refreshToken,
          user: authResponse.user,
          userID: authResponse.user?.id ?? authResponse.userID
        )
      }
    }

    if let wrapped = try? JSONDecoder().decode(DataEnvelope<AuthResponse>.self, from: data) {
      if !wrapped.data.token.isEmpty {
        return AuthResult(
          token: wrapped.data.token,
          refreshToken: wrapped.data.refreshToken,
          user: wrapped.data.user,
          userID: wrapped.data.user?.id ?? wrapped.data.userID
        )
      }
    }

    if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
      let token = extractString(
        from: json,
        paths: [
          ["token"],
          ["accessToken"],
          ["access_token"],
          ["session_token"],
          ["jwt"],
          ["data", "token"],
          ["data", "accessToken"],
          ["data", "access_token"],
          ["data", "session_token"],
          ["data", "jwt"],
          ["session", "token"],
          ["session", "accessToken"],
          ["session", "access_token"],
          ["session", "session_token"]
        ]
      )
      let refreshToken = extractString(
        from: json,
        paths: [
          ["refresh_token"],
          ["refreshToken"],
          ["data", "refresh_token"],
          ["data", "refreshToken"],
          ["session", "refresh_token"],
          ["session", "refreshToken"]
        ]
      )

      var user: User?
      var userID = extractString(
        from: json,
        paths: [
          ["userId"],
          ["user_id"],
          ["data", "userId"],
          ["data", "user_id"],
          ["session", "userId"],
          ["session", "user_id"]
        ]
      )

      if let userPayload = json["user"] {
        let userData = try JSONSerialization.data(withJSONObject: userPayload)
        user = try? JSONDecoder().decode(User.self, from: userData)
        userID = user?.id ?? userID
      }
      if user == nil,
         let dataUser = (json["data"] as? [String: Any])?["user"] {
        let userData = try JSONSerialization.data(withJSONObject: dataUser)
        user = try? JSONDecoder().decode(User.self, from: userData)
        userID = user?.id ?? userID
      }

      if let token, !token.isEmpty {
        return AuthResult(token: token, refreshToken: refreshToken, user: user, userID: userID)
      }
    }

    let payload = String(data: data, encoding: .utf8) ?? "<non-utf8 payload>"
    throw APIError.server(L10n.tr("api.error.auth_missing_token_payload", payload))
  }

  private func decodeObject<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
    let decoder = JSONDecoder()

    if let model = try? decoder.decode(T.self, from: data) {
      return model
    }

    if let wrapped = try? decoder.decode(DataEnvelope<T>.self, from: data) {
      return wrapped.data
    }

    throw APIError.decoding
  }

  private func debugPayloadSnippet(from data: Data, maxLength: Int = 1200) -> String {
#if DEBUG
    let text = String(data: data, encoding: .utf8) ?? "<non-utf8 payload>"
    if text.count <= maxLength {
      return text
    }
    let endIndex = text.index(text.startIndex, offsetBy: maxLength)
    return String(text[..<endIndex]) + "…"
#else
    return ""
#endif
  }

  private func debugLogAuthResponse(context: String, result: AuthResult, payload: Data) {
#if DEBUG
    let userDescription: String
    if let user = result.user {
      userDescription =
        "id=\(user.id) username=\(user.username) first=\(user.firstName ?? "nil") last=\(user.lastName ?? "nil")"
    } else {
      userDescription = "user=nil userID=\(result.userID ?? "nil")"
    }
    print(
      "[NoFrillzDebug][API][\(context)] tokenLength=\(result.token.count) refresh=\(result.refreshToken != nil) \(userDescription)"
    )
    print("[NoFrillzDebug][API][\(context)][payload] \(debugPayloadSnippet(from: payload))")
#endif
  }

  private func debugLogUserResponse(context: String, user: User, payload: Data) {
#if DEBUG
    print(
      "[NoFrillzDebug][API][\(context)] id=\(user.id) username=\(user.username) first=\(user.firstName ?? "nil") last=\(user.lastName ?? "nil")"
    )
    print("[NoFrillzDebug][API][\(context)][payload] \(debugPayloadSnippet(from: payload))")
#endif
  }

  private func debugLogFeedResponse(page: CursorPage<Post>, payload: Data, cursor: String?) {
#if DEBUG
    if let firstPost = page.items.first {
      print(
        "[NoFrillzDebug][API][fetchFeed] cursor=\(cursor ?? "nil") count=\(page.items.count) nextCursor=\(page.nextCursor ?? "nil") firstAuthorUsername=\(firstPost.authorUsername) first=\(firstPost.authorFirstName ?? "nil") last=\(firstPost.authorLastName ?? "nil")"
      )
    } else {
      print("[NoFrillzDebug][API][fetchFeed] cursor=\(cursor ?? "nil") count=0 nextCursor=\(page.nextCursor ?? "nil")")
    }
    print("[NoFrillzDebug][API][fetchFeed][payload] \(debugPayloadSnippet(from: payload))")
#endif
  }

  private func decodeList<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
    let decoder = JSONDecoder()

    if let model = try? decoder.decode(T.self, from: data) {
      return model
    }

    if let wrapped = try? decoder.decode(DataEnvelope<T>.self, from: data) {
      return wrapped.data
    }

    throw APIError.decoding
  }

  private func decodeUsersList(from data: Data, keys: [String] = ["users"]) throws -> [User] {
    try decodeUsersPage(from: data, keys: keys).items
  }

  private func decodeUsersPage(from data: Data, keys: [String] = ["users"]) throws -> CursorPage<User> {
    let decoder = JSONDecoder()

    if let wrapped = try? decoder.decode(UsersEnvelope.self, from: data) {
      return CursorPage(items: wrapped.users, nextCursor: wrapped.nextCursor)
    }

    if let users = try? decoder.decode([User].self, from: data) {
      return CursorPage(items: users, nextCursor: nil)
    }

    if let dataWrapped = try? decoder.decode(DataEnvelope<[User]>.self, from: data) {
      return CursorPage(items: dataWrapped.data, nextCursor: nil)
    }

    if let payload = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
      for key in keys {
        if let rawArray = payload[key] as? [Any] {
          let rawData = try JSONSerialization.data(withJSONObject: rawArray)
          if let users = try? decoder.decode([User].self, from: rawData) {
            return CursorPage(items: users, nextCursor: decodeNextCursor(from: payload))
          }
        }
      }
    }

    let payload = String(data: data, encoding: .utf8) ?? "<non-utf8 payload>"
    throw APIError.server(L10n.tr("api.error.decode_users_payload", payload))
  }

  private func decodePostsPage(from data: Data, listKeys: [String]) throws -> CursorPage<Post> {
    let decoder = JSONDecoder()

    if let wrapped = try? decoder.decode(PostsEnvelope.self, from: data) {
      return CursorPage(items: wrapped.posts, nextCursor: wrapped.nextCursor)
    }

    if let posts = try? decoder.decode([Post].self, from: data) {
      return CursorPage(items: posts, nextCursor: nil)
    }

    if let dataWrapped = try? decoder.decode(DataEnvelope<[Post]>.self, from: data) {
      return CursorPage(items: dataWrapped.data, nextCursor: nil)
    }

    if let payload = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
      for key in listKeys {
        if let rawArray = payload[key] as? [Any] {
          let rawData = try JSONSerialization.data(withJSONObject: rawArray)
          if let posts = try? decoder.decode([Post].self, from: rawData) {
            return CursorPage(items: posts, nextCursor: decodeNextCursor(from: payload))
          }
        }
      }
    }

    throw APIError.decoding
  }

  private func decodeBookmarksPage(from data: Data) throws -> CursorPage<Post> {
    let decoder = JSONDecoder()

    if let wrapped = try? decoder.decode(BookmarkedPostsEnvelope.self, from: data) {
      return CursorPage(items: wrapped.posts, nextCursor: wrapped.nextCursor)
    }

    if let bookmarks = try? decoder.decode([BookmarkedPost].self, from: data) {
      return CursorPage(items: bookmarks.map(\.post), nextCursor: nil)
    }

    if let dataWrapped = try? decoder.decode(DataEnvelope<[BookmarkedPost]>.self, from: data) {
      return CursorPage(items: dataWrapped.data.map(\.post), nextCursor: nil)
    }

    if let postsPage = try? decodePostsPage(from: data, listKeys: ["bookmarks", "posts"]) {
      let bookmarkedPosts = postsPage.items.map { $0.withBookmark(true, bookmarkedAt: $0.bookmarkedAt) }
      return CursorPage(items: bookmarkedPosts, nextCursor: postsPage.nextCursor)
    }

    if let payload = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
      for key in ["bookmarks", "items", "posts"] {
        guard let rawArray = payload[key] as? [Any] else { continue }
        let rawData = try JSONSerialization.data(withJSONObject: rawArray)

        if let bookmarks = try? decoder.decode([BookmarkedPost].self, from: rawData) {
          return CursorPage(items: bookmarks.map(\.post), nextCursor: decodeNextCursor(from: payload))
        }

        if let posts = try? decoder.decode([Post].self, from: rawData) {
          return CursorPage(
            items: posts.map { $0.withBookmark(true, bookmarkedAt: $0.bookmarkedAt) },
            nextCursor: decodeNextCursor(from: payload)
          )
        }
      }
    }

    throw APIError.decoding
  }

  private func decodeTopicsList(from data: Data, keys: [String] = ["topics"]) throws -> [Topic] {
    let decoder = JSONDecoder()

    if let wrapped = try? decoder.decode(TopicsEnvelope.self, from: data) {
      return wrapped.topics
    }

    if let topics = try? decoder.decode([Topic].self, from: data) {
      return topics
    }

    if let dataWrapped = try? decoder.decode(DataEnvelope<[Topic]>.self, from: data) {
      return dataWrapped.data
    }

    if let payload = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
      for key in keys {
        if let rawArray = payload[key] as? [Any] {
          let rawData = try JSONSerialization.data(withJSONObject: rawArray)
          if let topics = try? decoder.decode([Topic].self, from: rawData) {
            return topics
          }
        }
      }
    }

    throw APIError.decoding
  }

  private func decodeIDList(from data: Data) throws -> [String] {
    if let values = try? JSONDecoder().decode([String].self, from: data) {
      return values
    }
    if let values = try? JSONDecoder().decode([Int64].self, from: data) {
      return values.map(String.init)
    }
    if let values = try? JSONDecoder().decode([UInt64].self, from: data) {
      return values.map(String.init)
    }
    if let values = try? JSONDecoder().decode([Double].self, from: data) {
      return values.map { String(Int64($0)) }
    }

    if let payload = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
      for key in ["following", "ids", "users"] {
        guard let rawArray = payload[key] as? [Any] else { continue }
        let values = rawArray.compactMap { value -> String? in
          if let string = value as? String, !string.isEmpty {
            return string
          }
          if let number = value as? NSNumber {
            return number.stringValue
          }
          return nil
        }
        if !values.isEmpty {
          return values
        }
      }
    }

    throw APIError.decoding
  }

  private func extractMessage(from data: Data) throws -> String {
    let decoder = JSONDecoder()

    if let messageContainer = try? decoder.decode(MessageEnvelope.self, from: data) {
      return messageContainer.message
    }

    let generic = try JSONSerialization.jsonObject(with: data) as? [String: Any]
    return (generic?["error"] as? String)
      ?? (generic?["message"] as? String)
      ?? L10n.tr("api.error.server_request_failed")
  }
}

private func extractString(from payload: [String: Any], paths: [[String]]) -> String? {
  for path in paths {
    if let value = value(in: payload, path: path) as? String, !value.isEmpty {
      return value
    }
  }
  return nil
}

private func value(in payload: [String: Any], path: [String]) -> Any? {
  guard !path.isEmpty else { return nil }
  var current: Any = payload

  for key in path {
    guard let dict = current as? [String: Any], let next = dict[key] else {
      return nil
    }
    current = next
  }

  return current
}

private func decodeNextCursor(from payload: [String: Any]) -> String? {
  if let cursor = payload["next_cursor"] as? String, !cursor.isEmpty {
    return cursor
  }
  if let cursor = payload["nextCursor"] as? String, !cursor.isEmpty {
    return cursor
  }
  return nil
}

private struct DataEnvelope<T: Decodable>: Decodable {
  let data: T
}

private struct UsersEnvelope: Decodable {
  let users: [User]
  let nextCursor: String?

  enum CodingKeys: String, CodingKey {
    case users
    case nextCursor = "next_cursor"
  }
}

private struct TopicsEnvelope: Decodable {
  let topics: [Topic]
}

private struct MessageEnvelope: Decodable {
  let message: String
}

private struct AuthResponse: Decodable {
  let token: String
  let refreshToken: String?
  let user: User?
  let userID: String?

  enum CodingKeys: String, CodingKey {
    case token
    case accessToken
    case accessTokenSnake = "access_token"
    case sessionToken = "session_token"
    case refreshTokenKey = "refresh_token"
    case refreshTokenCamel = "refreshToken"
    case jwt
    case user
    case userID = "userId"
    case userIDSnake = "user_id"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    token = try container.decodeIfPresent(String.self, forKey: .token)
      ?? container.decodeIfPresent(String.self, forKey: .accessToken)
      ?? container.decodeIfPresent(String.self, forKey: .accessTokenSnake)
      ?? container.decodeIfPresent(String.self, forKey: .sessionToken)
      ?? container.decodeIfPresent(String.self, forKey: .jwt)
      ?? ""
    refreshToken = try container.decodeIfPresent(String.self, forKey: .refreshTokenKey)
      ?? container.decodeIfPresent(String.self, forKey: .refreshTokenCamel)
    user = try container.decodeIfPresent(User.self, forKey: .user)
    userID = try container.decodeIfPresent(String.self, forKey: .userID)
      ?? container.decodeIfPresent(String.self, forKey: .userIDSnake)
  }
}

private struct FeedResponse: Decodable {
  let posts: [Post]
  let nextCursor: String?

  enum CodingKeys: String, CodingKey {
    case posts
    case nextCursor = "next_cursor"
  }
}

private struct PostsEnvelope: Decodable {
  let posts: [Post]
  let nextCursor: String?

  enum CodingKeys: String, CodingKey {
    case posts
    case nextCursor = "next_cursor"
  }
}

private struct BookmarkedPostsEnvelope: Decodable {
  let posts: [Post]
  let nextCursor: String?

  enum CodingKeys: String, CodingKey {
    case bookmarks
    case posts
    case items
    case nextCursor = "next_cursor"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    let decodedPosts = try container.decodeIfPresent([BookmarkedPost].self, forKey: .bookmarks)
      ?? container.decodeIfPresent([BookmarkedPost].self, forKey: .items)
      ?? container.decodeIfPresent([BookmarkedPost].self, forKey: .posts)
      ?? []

    posts = decodedPosts.map(\.post)
    nextCursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
  }
}

private struct BookmarkedPost: Decodable {
  let post: Post

  private enum CodingKeys: String, CodingKey {
    case post
    case data
    case bookmarkedAtSnake = "bookmarked_at"
    case bookmarkedAtCamel = "bookmarkedAt"
  }

  init(from decoder: Decoder) throws {
    if let directPost = try? Post(from: decoder) {
      post = directPost.withBookmark(true, bookmarkedAt: directPost.bookmarkedAt)
      return
    }

    let container = try decoder.container(keyedBy: CodingKeys.self)
    let nestedPost = try container.decodeIfPresent(Post.self, forKey: .post)
      ?? container.decodeIfPresent(Post.self, forKey: .data)

    guard let nestedPost else {
      throw DecodingError.dataCorruptedError(
        forKey: .post,
        in: container,
        debugDescription: "Missing bookmark post payload"
      )
    }

    let bookmarkedAtString = try container.decodeIfPresent(String.self, forKey: .bookmarkedAtSnake)
      ?? container.decodeIfPresent(String.self, forKey: .bookmarkedAtCamel)
    let bookmarkedAt = bookmarkedAtString.flatMap { DateParsers.apiDate(from: $0) }
    post = nestedPost.withBookmark(true, bookmarkedAt: bookmarkedAt ?? nestedPost.bookmarkedAt)
  }
}

private struct CreatePostRequest: Encodable {
  let body: String
}

private struct SignupRequest: Encodable {
  let firstName: String
  let lastName: String
  let username: String
  let email: String
  let password: String
  let about: String?

  enum CodingKeys: String, CodingKey {
    case firstName = "first_name"
    case lastName = "last_name"
    case username
    case email
    case password
    case about
  }
}

private struct UpdateCurrentUserRequest: Encodable {
  let firstName: String
  let lastName: String
  let email: String
  let about: String?

  enum CodingKeys: String, CodingKey {
    case firstName = "first_name"
    case lastName = "last_name"
    case email
    case about
  }
}

private struct ChangePasswordRequest: Encodable {
  let currentPassword: String
  let newPassword: String

  enum CodingKeys: String, CodingKey {
    case currentPassword = "current_password"
    case newPassword = "new_password"
  }
}

private struct RegisterPushDeviceRequest: Encodable {
  let token: String
}

private struct AnyEncodable: Encodable {
  private let encodeBlock: (Encoder) throws -> Void

  init(_ value: any Encodable) {
    encodeBlock = value.encode
  }

  func encode(to encoder: Encoder) throws {
    try encodeBlock(encoder)
  }
}
