import Foundation

struct User: Decodable, Equatable {
  let id: String
  let username: String
  let firstName: String?
  let lastName: String?
  let email: String?
  let about: String?
  let avatarURL: String?
  let joinedAt: Date?
  let isFollowing: Bool
  let isFollower: Bool
  let followerCount: Int
  let followingCount: Int
  let postCount: Int

  init(
    id: String,
    username: String,
    firstName: String? = nil,
    lastName: String? = nil,
    email: String?,
    about: String?,
    avatarURL: String? = nil,
    joinedAt: Date?,
    isFollowing: Bool = false,
    isFollower: Bool = false,
    followerCount: Int = 0,
    followingCount: Int = 0,
    postCount: Int = 0
  ) {
    self.id = id
    self.username = username
    self.firstName = firstName
    self.lastName = lastName
    self.email = email
    self.about = about
    self.avatarURL = avatarURL
    self.joinedAt = joinedAt
    self.isFollowing = isFollowing
    self.isFollower = isFollower
    self.followerCount = followerCount
    self.followingCount = followingCount
    self.postCount = postCount
  }

  enum CodingKeys: String, CodingKey {
    case id
    case userID = "userId"
    case userIDSnake = "user_id"
    case username
    case firstNameSnake = "first_name"
    case firstNameCamel = "firstName"
    case lastNameSnake = "last_name"
    case lastNameCamel = "lastName"
    case handle
    case email
    case about
    case avatarURLSnake = "avatar_url"
    case avatarURLCamel = "avatarURL"
    case avatarURLLowerCamel = "avatarUrl"
    case joinedAt
    case createdAt
    case isFollowingSnake = "is_following"
    case isFollowingCamel = "isFollowing"
    case isFollowerSnake = "is_follower"
    case isFollowerCamel = "isFollower"
    case followerCountSnake = "follower_count"
    case followerCountCamel = "followerCount"
    case followingCountSnake = "following_count"
    case followingCountCamel = "followingCount"
    case postCountSnake = "post_count"
    case postCountCamel = "postCount"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    guard let rawID = try DecodingHelpers.decodeID(container: container, keys: [.id, .userID, .userIDSnake]) else {
      throw DecodingError.dataCorruptedError(forKey: .id, in: container, debugDescription: "Missing user ID")
    }

    let rawUsername = try container.decodeIfPresent(String.self, forKey: .username)
      ?? container.decodeIfPresent(String.self, forKey: .handle)
      ?? L10n.tr("user.fallback.anonymous")

    id = rawID
    username = rawUsername
    firstName = try container.decodeIfPresent(String.self, forKey: .firstNameSnake)
      ?? container.decodeIfPresent(String.self, forKey: .firstNameCamel)
    lastName = try container.decodeIfPresent(String.self, forKey: .lastNameSnake)
      ?? container.decodeIfPresent(String.self, forKey: .lastNameCamel)
    email = try container.decodeIfPresent(String.self, forKey: .email)
    about = try container.decodeIfPresent(String.self, forKey: .about)
    avatarURL = try container.decodeIfPresent(String.self, forKey: .avatarURLSnake)
      ?? container.decodeIfPresent(String.self, forKey: .avatarURLCamel)
      ?? container.decodeIfPresent(String.self, forKey: .avatarURLLowerCamel)

    if let joinedAtString = try container.decodeIfPresent(String.self, forKey: .joinedAt)
      ?? container.decodeIfPresent(String.self, forKey: .createdAt) {
      joinedAt = DateParsers.apiDate(from: joinedAtString)
    } else {
      joinedAt = nil
    }

    let isFollowingSnake = try container.decodeIfPresent(Bool.self, forKey: .isFollowingSnake)
    let isFollowingCamel = try container.decodeIfPresent(Bool.self, forKey: .isFollowingCamel)
    isFollowing = isFollowingSnake ?? isFollowingCamel ?? false

    let isFollowerSnake = try container.decodeIfPresent(Bool.self, forKey: .isFollowerSnake)
    let isFollowerCamel = try container.decodeIfPresent(Bool.self, forKey: .isFollowerCamel)
    isFollower = isFollowerSnake ?? isFollowerCamel ?? false
    followerCount = try DecodingHelpers.decodeInt(
      container: container,
      keys: [.followerCountSnake, .followerCountCamel]
    ) ?? 0
    followingCount = try DecodingHelpers.decodeInt(
      container: container,
      keys: [.followingCountSnake, .followingCountCamel]
    ) ?? 0
    postCount = try DecodingHelpers.decodeInt(
      container: container,
      keys: [.postCountSnake, .postCountCamel]
    ) ?? 0
  }

  var fullName: String? {
    let first = firstName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    let last = lastName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    let combined = "\(first) \(last)".trimmingCharacters(in: .whitespacesAndNewlines)
    return combined.isEmpty ? nil : combined
  }

  var displayName: String {
    fullName ?? username
  }

  var initials: String {
    let firstInitial = firstName?
      .trimmingCharacters(in: .whitespacesAndNewlines)
      .prefix(1) ?? ""
    let lastInitial = lastName?
      .trimmingCharacters(in: .whitespacesAndNewlines)
      .prefix(1) ?? ""
    let combined = "\(firstInitial)\(lastInitial)"
    if combined.isEmpty {
      return String(username.prefix(1)).lowercased()
    }
    return combined.lowercased()
  }

  func withFollowing(_ isFollowing: Bool) -> User {
    User(
      id: id,
      username: username,
      firstName: firstName,
      lastName: lastName,
      email: email,
      about: about,
      avatarURL: avatarURL,
      joinedAt: joinedAt,
      isFollowing: isFollowing,
      isFollower: isFollower,
      followerCount: followerCount,
      followingCount: followingCount,
      postCount: postCount
    )
  }

  func withRelationship(isFollowing: Bool, followerCount: Int) -> User {
    User(
      id: id,
      username: username,
      firstName: firstName,
      lastName: lastName,
      email: email,
      about: about,
      avatarURL: avatarURL,
      joinedAt: joinedAt,
      isFollowing: isFollowing,
      isFollower: isFollower,
      followerCount: followerCount,
      followingCount: followingCount,
      postCount: postCount
    )
  }

  func withFollowingCount(_ followingCount: Int) -> User {
    User(
      id: id,
      username: username,
      firstName: firstName,
      lastName: lastName,
      email: email,
      about: about,
      avatarURL: avatarURL,
      joinedAt: joinedAt,
      isFollowing: isFollowing,
      isFollower: isFollower,
      followerCount: followerCount,
      followingCount: followingCount,
      postCount: postCount
    )
  }
}

struct Topic: Decodable, Equatable {
  let name: String
  let description: String
  let imageURL: String?

  enum CodingKeys: String, CodingKey {
    case name
    case description
    case image
    case imageURLSnake = "image_url"
    case imageURLCamel = "imageURL"
    case imageURLLowerCamel = "imageUrl"
  }

  init(name: String, description: String, imageURL: String? = nil) {
    self.name = name
    self.description = description
    self.imageURL = imageURL
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    name = try container.decode(String.self, forKey: .name)
    description = try container.decode(String.self, forKey: .description)
    imageURL = try container.decodeIfPresent(String.self, forKey: .image)
      ?? container.decodeIfPresent(String.self, forKey: .imageURLSnake)
      ?? container.decodeIfPresent(String.self, forKey: .imageURLCamel)
      ?? container.decodeIfPresent(String.self, forKey: .imageURLLowerCamel)
  }
}

struct Post: Decodable, Equatable {
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

  enum CodingKeys: String, CodingKey {
    case id
    case postID = "postId"
    case postIDSnake = "post_id"
    case liked
    case likedSnake = "is_liked"
    case user
    case body
    case createdAt
    case created
    case url
    case source
    case likeCountSnake = "like_count"
    case likeCountCamel = "likeCount"
    case commentCountSnake = "comment_count"
    case commentCountCamel = "commentCount"
    case isBookmarkedSnake = "is_bookmarked"
    case isBookmarkedCamel = "isBookmarked"
    case bookmarkedAtSnake = "bookmarked_at"
    case bookmarkedAtCamel = "bookmarkedAt"
  }

  private enum UserCodingKeys: String, CodingKey {
    case id
    case userID = "user_id"
    case username
    case firstName = "first_name"
    case lastName = "last_name"
    case avatarURL = "avatar_url"
  }

  init(
    id: String,
    url: String? = nil,
    authorID: String,
    authorUsername: String,
    authorFirstName: String? = nil,
    authorLastName: String? = nil,
    authorAvatarURL: String? = nil,
    content: String,
    createdAt: Date?,
    source: String = "human",
    liked: Bool = false,
    isBookmarked: Bool = false,
    bookmarkedAt: Date? = nil,
    likeCount: Int,
    commentCount: Int
  ) {
    self.id = id
    self.url = url
    self.authorID = authorID
    self.authorUsername = authorUsername
    self.authorFirstName = authorFirstName
    self.authorLastName = authorLastName
    self.authorAvatarURL = authorAvatarURL
    self.content = content
    self.createdAt = createdAt
    self.source = source
    self.liked = liked
    self.isBookmarked = isBookmarked
    self.bookmarkedAt = bookmarkedAt
    self.likeCount = likeCount
    self.commentCount = commentCount
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)

    guard let rawID = try DecodingHelpers.decodeID(container: container, keys: [.id, .postID, .postIDSnake]) else {
      throw DecodingError.dataCorruptedError(forKey: .id, in: container, debugDescription: "Missing post ID")
    }
    id = rawID
    url = try container.decodeIfPresent(String.self, forKey: .url)

    content = try container.decode(String.self, forKey: .body)

    let userContainer = try container.nestedContainer(keyedBy: UserCodingKeys.self, forKey: .user)
    guard let rawAuthorID = try DecodingHelpers.decodeID(container: userContainer, keys: [.id, .userID]) else {
      throw DecodingError.dataCorruptedError(forKey: .user, in: container, debugDescription: "Missing post author ID")
    }
    authorID = rawAuthorID
    authorUsername = try userContainer.decode(String.self, forKey: .username)
    authorFirstName = try userContainer.decode(String.self, forKey: .firstName)
    authorLastName = try userContainer.decode(String.self, forKey: .lastName)
    authorAvatarURL = try userContainer.decodeIfPresent(String.self, forKey: .avatarURL)

    if let createdAtString = try container.decodeIfPresent(String.self, forKey: .createdAt)
      ?? container.decodeIfPresent(String.self, forKey: .created) {
      createdAt = DateParsers.apiDate(from: createdAtString)
    } else {
      createdAt = nil
    }
    source = (try container.decodeIfPresent(String.self, forKey: .source) ?? "human").lowercased()

    liked = try container.decodeIfPresent(Bool.self, forKey: .liked)
      ?? container.decodeIfPresent(Bool.self, forKey: .likedSnake)
      ?? false
    isBookmarked = try container.decodeIfPresent(Bool.self, forKey: .isBookmarkedSnake)
      ?? container.decodeIfPresent(Bool.self, forKey: .isBookmarkedCamel)
      ?? false
    if let bookmarkedAtString = try container.decodeIfPresent(String.self, forKey: .bookmarkedAtSnake)
      ?? container.decodeIfPresent(String.self, forKey: .bookmarkedAtCamel) {
      bookmarkedAt = DateParsers.apiDate(from: bookmarkedAtString)
    } else {
      bookmarkedAt = nil
    }
    likeCount = try DecodingHelpers.decodeInt(container: container, keys: [.likeCountSnake, .likeCountCamel]) ?? 0
    commentCount = try DecodingHelpers.decodeInt(container: container, keys: [.commentCountSnake, .commentCountCamel]) ?? 0
  }

  var authorDisplayName: String {
    let first = authorFirstName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    let last = authorLastName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    let combined = "\(first) \(last)".trimmingCharacters(in: .whitespacesAndNewlines)
    if !combined.isEmpty {
      return combined
    }
    return authorUsername
  }

  var isAI: Bool {
    source == "ai"
  }

  func withLiked(_ nextLiked: Bool) -> Post {
    let updatedLikeCount: Int
    if nextLiked == liked {
      updatedLikeCount = likeCount
    } else if nextLiked {
      updatedLikeCount = likeCount + 1
    } else {
      updatedLikeCount = max(0, likeCount - 1)
    }

    return Post(
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
      liked: nextLiked,
      isBookmarked: isBookmarked,
      bookmarkedAt: bookmarkedAt,
      likeCount: updatedLikeCount,
      commentCount: commentCount
    )
  }

  func withBookmark(_ nextIsBookmarked: Bool, bookmarkedAt nextBookmarkedAt: Date?) -> Post {
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
      isBookmarked: nextIsBookmarked,
      bookmarkedAt: nextIsBookmarked ? nextBookmarkedAt : nil,
      likeCount: likeCount,
      commentCount: commentCount
    )
  }
}

struct PostPreview {
  let text: String
  let isTruncated: Bool
}

extension Post {
  private static let estimatedReadingWordsPerMinute = 238.0
  private static let feedPreviewCharacterLimit = 520

  var estimatedReadingMinutes: Int {
    let wordCount = max(
      1,
      content.split(whereSeparator: { $0.isWhitespace }).count
    )
    return max(1, Int(ceil(Double(wordCount) / Self.estimatedReadingWordsPerMinute)))
  }

  var estimatedReadingTimeText: String {
    "\(estimatedReadingMinutes) min read"
  }

  var displayEstimatedReadingTimeText: String? {
    estimatedReadingMinutes > 1 ? estimatedReadingTimeText : nil
  }

  var feedPreview: PostPreview {
    preview(maxCharacters: Self.feedPreviewCharacterLimit)
  }

  func excerpt(maxCharacters: Int) -> String {
    preview(maxCharacters: maxCharacters).text
  }

  private func preview(maxCharacters: Int) -> PostPreview {
    let normalized = content.trimmingCharacters(in: .whitespacesAndNewlines)
    guard normalized.count > maxCharacters else {
      return PostPreview(text: normalized, isTruncated: false)
    }

    let endIndex = normalized.index(normalized.startIndex, offsetBy: maxCharacters)
    let rawPrefix = String(normalized[..<endIndex])
    let searchWindowSize = min(80, rawPrefix.count)
    let searchStartIndex = rawPrefix.index(rawPrefix.endIndex, offsetBy: -searchWindowSize)
    let boundarySearchSlice = rawPrefix[searchStartIndex...]

    let preferredBoundary = boundarySearchSlice.lastIndex(where: {
      $0.isWhitespace || $0 == "." || $0 == "," || $0 == ";" || $0 == "!" || $0 == "?"
    })

    let trimmedPrefix: String
    if let preferredBoundary {
      let candidate = String(rawPrefix[..<preferredBoundary]).trimmingCharacters(in: .whitespacesAndNewlines)
      trimmedPrefix = candidate.isEmpty ? rawPrefix.trimmingCharacters(in: .whitespacesAndNewlines) : candidate
    } else {
      trimmedPrefix = rawPrefix.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    return PostPreview(text: "\(trimmedPrefix)…", isTruncated: true)
  }
}

struct AuthResult: Equatable {
  let token: String
  let refreshToken: String?
  let user: User?
  let userID: String?
}

enum DateParsers {
  private static let iso8601WithFractionalSeconds: ISO8601DateFormatter = {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    return formatter
  }()

  private static let iso8601: ISO8601DateFormatter = {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return formatter
  }()

  static func apiDate(from value: String) -> Date? {
    if let parsed = iso8601WithFractionalSeconds.date(from: value) {
      return parsed
    }
    return iso8601.date(from: value)
  }
}

enum DecodingHelpers {
  static func decodeID<K: CodingKey>(container: KeyedDecodingContainer<K>, keys: [K]) throws -> String? {
    for key in keys {
      if let stringValue = try? container.decode(String.self, forKey: key),
         !stringValue.isEmpty {
        return stringValue
      }
      if let intValue = try? container.decode(Int64.self, forKey: key) {
        return String(intValue)
      }
      if let uintValue = try? container.decode(UInt64.self, forKey: key) {
        return String(uintValue)
      }
      if let doubleValue = try? container.decode(Double.self, forKey: key) {
        return String(Int64(doubleValue))
      }
    }
    return nil
  }

  static func decodeInt<K: CodingKey>(container: KeyedDecodingContainer<K>, keys: [K]) throws -> Int? {
    for key in keys {
      if let intValue = try? container.decode(Int.self, forKey: key) {
        return intValue
      }
      if let int64Value = try? container.decode(Int64.self, forKey: key) {
        return Int(int64Value)
      }
      if let uint64Value = try? container.decode(UInt64.self, forKey: key) {
        return Int(uint64Value)
      }
      if let doubleValue = try? container.decode(Double.self, forKey: key) {
        return Int(doubleValue)
      }
      if let stringValue = try? container.decode(String.self, forKey: key),
         let parsed = Int(stringValue) {
        return parsed
      }
    }
    return nil
  }
}

extension DateFormatter {
  static let feedDate: DateFormatter = {
    let formatter = DateFormatter()
    formatter.dateStyle = .medium
    formatter.timeStyle = .short
    return formatter
  }()

  static let monthYearDate: DateFormatter = {
    let formatter = DateFormatter()
    formatter.dateFormat = "MMM yyyy"
    return formatter
  }()

  static let yearDate: DateFormatter = {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy"
    return formatter
  }()
}

enum PostDateDisplay {
  private static let relativeFormatter: RelativeDateTimeFormatter = {
    let formatter = RelativeDateTimeFormatter()
    formatter.unitsStyle = .full
    formatter.dateTimeStyle = .named
    return formatter
  }()

  static func string(from date: Date) -> String {
    let now = Date()
    let age = now.timeIntervalSince(date)
    if age >= 0, age < 60 {
      return L10n.tr("date.now")
    }
    if age >= 0, age < 60 * 5 {
      return L10n.tr("date.few_minutes_ago")
    }
    if age >= 0, age < 60 * 60 {
      let minutes = max(1, Int(age / 60))
      if minutes == 1 {
        return L10n.tr("date.one_min_ago")
      }
      return L10n.tr("date.mins_ago", minutes)
    }
    if age >= 0, age < 60 * 60 * 24 * 365 {
      return relativeFormatter.localizedString(for: date, relativeTo: now)
    }
    if age >= 0, age < 60 * 60 * 24 * 365 * 3 {
      return DateFormatter.monthYearDate.string(from: date)
    }
    return DateFormatter.yearDate.string(from: date)
  }

  static func shortString(from date: Date) -> String {
    let age = max(0, Date().timeIntervalSince(date))

    if age < 60 {
      return L10n.tr("date.short.now")
    }
    if age < 60 * 60 {
      return L10n.tr("date.short.min", max(1, Int(age / 60)))
    }
    if age < 60 * 60 * 24 {
      return L10n.tr("date.short.hour", max(1, Int(age / (60 * 60))))
    }
    if age < 60 * 60 * 24 * 7 {
      return L10n.tr("date.short.day", max(1, Int(age / (60 * 60 * 24))))
    }
    if age < 60 * 60 * 24 * 30 {
      return L10n.tr("date.short.week", max(1, Int(age / (60 * 60 * 24 * 7))))
    }
    if age < 60 * 60 * 24 * 365 {
      return L10n.tr("date.short.month", max(1, Int(age / (60 * 60 * 24 * 30))))
    }
    return L10n.tr("date.short.year", max(1, Int(age / (60 * 60 * 24 * 365))))
  }
}
