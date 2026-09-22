import Foundation

@MainActor
final class BookmarkStore {
  private enum Constants {
    static let cacheKeyPrefix = "nofrillz.readingList."
  }

  private struct BookmarkState: Equatable {
    let isBookmarked: Bool
    let bookmarkedAt: Date?
  }

  private struct CachedReadingListPayload: Codable {
    let posts: [CachedPost]
  }

  private struct CachedPost: Codable {
    let id: String
    let url: String?
    let authorID: String
    let authorUsername: String
    let authorFirstName: String?
    let authorLastName: String?
    let authorAvatarURL: String?
    let content: String
    let createdAt: Date?
    let source: String
    let liked: Bool
    let isBookmarked: Bool
    let bookmarkedAt: Date?
    let likeCount: Int
    let commentCount: Int

    init(post: Post) {
      id = post.id
      url = post.url
      authorID = post.authorID
      authorUsername = post.authorUsername
      authorFirstName = post.authorFirstName
      authorLastName = post.authorLastName
      authorAvatarURL = post.authorAvatarURL
      content = post.content
      createdAt = post.createdAt
      source = post.source
      liked = post.liked
      isBookmarked = post.isBookmarked
      bookmarkedAt = post.bookmarkedAt
      likeCount = post.likeCount
      commentCount = post.commentCount
    }

    var post: Post {
      Post(
        id: id,
        url: url,
        authorID: authorID,
        authorUsername: authorUsername,
        authorFirstName: authorFirstName,
        authorLastName: authorLastName,
        authorAvatarURL: authorAvatarURL,
        content: content,
        createdAt: createdAt,
        source: source,
        liked: liked,
        isBookmarked: isBookmarked,
        bookmarkedAt: bookmarkedAt,
        likeCount: likeCount,
        commentCount: commentCount
      )
    }
  }

  private let api: NoFrillzAPI
  private let sessionStore: SessionStore
  private let defaults: UserDefaults

  private var bookmarkStates: [String: BookmarkState] = [:]
  private var inFlightPostIDs = Set<String>()
  private var readingListPosts: [Post] = []
  private var cachedUserID: String?

  init(api: NoFrillzAPI, sessionStore: SessionStore, defaults: UserDefaults = .standard) {
    self.api = api
    self.sessionStore = sessionStore
    self.defaults = defaults
    restoreStateForCurrentSession()
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleSessionDidChange),
      name: .sessionDidChange,
      object: nil
    )
  }

  deinit {
    NotificationCenter.default.removeObserver(self)
  }

  func displayedPost(from post: Post) -> Post {
    guard let state = bookmarkStates[post.id] else { return post }
    return post.withBookmark(state.isBookmarked, bookmarkedAt: state.bookmarkedAt)
  }

  func absorbServerPosts(_ posts: [Post]) -> [Post] {
    var didChangeCache = false
    let resolvedPosts = posts.map { post in
      if !inFlightPostIDs.contains(post.id) {
        bookmarkStates[post.id] = BookmarkState(
          isBookmarked: post.isBookmarked,
          bookmarkedAt: post.bookmarkedAt
        )
        didChangeCache = reconcileReadingListCache(
          with: post,
          insertingAtFront: false
        ) || didChangeCache
      }
      return displayedPost(from: post)
    }

    if didChangeCache {
      persistReadingListCache()
    }

    return resolvedPosts
  }

  func absorbReadingListPosts(_ posts: [Post], reset: Bool) -> [Post] {
    let resolvedPosts = posts.compactMap { post -> Post? in
      let normalized = post.withBookmark(true, bookmarkedAt: post.bookmarkedAt)
      if !inFlightPostIDs.contains(normalized.id) {
        bookmarkStates[normalized.id] = BookmarkState(
          isBookmarked: true,
          bookmarkedAt: normalized.bookmarkedAt
        )
      }

      let displayed = displayedPost(from: normalized)
      return displayed.isBookmarked ? displayed : nil
    }

    if reset {
      readingListPosts = deduplicatedPosts(resolvedPosts)
    } else {
      mergeReadingListCache(with: resolvedPosts)
    }

    persistReadingListCache()
    return resolvedPosts
  }

  func cachedReadingListPosts() -> [Post] {
    readingListPosts
      .map { displayedPost(from: $0) }
      .filter(\.isBookmarked)
  }

  var hasCachedReadingListPosts: Bool {
    !cachedReadingListPosts().isEmpty
  }

  func isMutationInFlight(for postID: String) -> Bool {
    inFlightPostIDs.contains(postID)
  }

  func setBookmarked(_ isBookmarked: Bool, for post: Post) async throws -> Post {
    let currentPost = displayedPost(from: post)
    guard !inFlightPostIDs.contains(currentPost.id) else { return currentPost }

    let hadStoredState = bookmarkStates[currentPost.id] != nil
    let previousState = bookmarkStates[currentPost.id] ?? BookmarkState(
      isBookmarked: currentPost.isBookmarked,
      bookmarkedAt: currentPost.bookmarkedAt
    )
    let nextBookmarkedAt = isBookmarked ? (currentPost.bookmarkedAt ?? Date()) : nil
    let optimisticState = BookmarkState(isBookmarked: isBookmarked, bookmarkedAt: nextBookmarkedAt)
    let optimisticPost = currentPost.withBookmark(isBookmarked, bookmarkedAt: nextBookmarkedAt)

    inFlightPostIDs.insert(currentPost.id)
    bookmarkStates[currentPost.id] = optimisticState
    _ = reconcileReadingListCache(with: optimisticPost, insertingAtFront: isBookmarked)
    persistReadingListCache()
    postBookmarkChange(for: optimisticPost)

    do {
      if isBookmarked {
        try await api.bookmarkPost(id: currentPost.id)
      } else {
        try await api.removeBookmark(id: currentPost.id)
      }

      inFlightPostIDs.remove(currentPost.id)
      persistReadingListCache()
      postBookmarkChange(for: optimisticPost)
      return optimisticPost
    } catch {
      inFlightPostIDs.remove(currentPost.id)

      if hadStoredState {
        bookmarkStates[currentPost.id] = previousState
      } else {
        bookmarkStates.removeValue(forKey: currentPost.id)
      }

      let rollbackPost = currentPost.withBookmark(
        previousState.isBookmarked,
        bookmarkedAt: previousState.bookmarkedAt
      )
      _ = reconcileReadingListCache(with: rollbackPost, insertingAtFront: false)
      persistReadingListCache()
      postBookmarkChange(for: rollbackPost)
      throw error
    }
  }

  @objc private func handleSessionDidChange() {
    restoreStateForCurrentSession()
  }

  private func restoreStateForCurrentSession() {
    clearInMemoryState()

    guard let userID = sessionStore.currentUser?.id, !userID.isEmpty else { return }
    cachedUserID = userID

    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601

    guard let data = defaults.data(forKey: cacheKey(for: userID)),
          let payload = try? decoder.decode(CachedReadingListPayload.self, from: data) else {
      return
    }

    readingListPosts = payload.posts.map(\.post)
    for post in readingListPosts where post.isBookmarked {
      bookmarkStates[post.id] = BookmarkState(
        isBookmarked: true,
        bookmarkedAt: post.bookmarkedAt
      )
    }
  }

  private func clearInMemoryState() {
    bookmarkStates.removeAll()
    inFlightPostIDs.removeAll()
    readingListPosts.removeAll()
    cachedUserID = nil
  }

  private func reconcileReadingListCache(with post: Post, insertingAtFront: Bool) -> Bool {
    let resolvedPost = displayedPost(from: post)
    let previousPosts = readingListPosts

    if resolvedPost.isBookmarked {
      if let existingIndex = readingListPosts.firstIndex(where: { $0.id == resolvedPost.id }) {
        let updatedIndex = insertingAtFront ? 0 : existingIndex
        readingListPosts.remove(at: existingIndex)
        readingListPosts.insert(resolvedPost, at: updatedIndex)
      } else if insertingAtFront {
        readingListPosts.insert(resolvedPost, at: 0)
      } else {
        readingListPosts.append(resolvedPost)
      }
    } else {
      readingListPosts.removeAll { $0.id == resolvedPost.id }
    }

    return readingListPosts != previousPosts
  }

  private func mergeReadingListCache(with posts: [Post]) {
    for post in posts {
      if let index = readingListPosts.firstIndex(where: { $0.id == post.id }) {
        readingListPosts[index] = post
      } else {
        readingListPosts.append(post)
      }
    }

    readingListPosts = deduplicatedPosts(readingListPosts.filter(\.isBookmarked))
  }

  private func deduplicatedPosts(_ posts: [Post]) -> [Post] {
    var seenIDs = Set<String>()
    var deduplicated: [Post] = []
    deduplicated.reserveCapacity(posts.count)

    for post in posts where seenIDs.insert(post.id).inserted {
      deduplicated.append(post)
    }

    return deduplicated
  }

  private func persistReadingListCache() {
    guard let userID = cachedUserID ?? sessionStore.currentUser?.id, !userID.isEmpty else { return }

    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601

    let payload = CachedReadingListPayload(posts: cachedReadingListPosts().map(CachedPost.init))
    guard let data = try? encoder.encode(payload) else { return }
    defaults.set(data, forKey: cacheKey(for: userID))
  }

  private func cacheKey(for userID: String) -> String {
    "\(Constants.cacheKeyPrefix)\(userID)"
  }

  private func postBookmarkChange(for post: Post) {
    NotificationCenter.default.post(
      name: .bookmarkStateDidChange,
      object: BookmarkStateChange(
        post: displayedPost(from: post),
        isInFlight: inFlightPostIDs.contains(post.id)
      )
    )
  }
}
