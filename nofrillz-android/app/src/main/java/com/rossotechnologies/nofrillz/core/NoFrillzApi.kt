package com.rossotechnologies.nofrillz.core

import com.google.gson.Gson
import com.google.gson.JsonArray
import com.google.gson.JsonElement
import com.google.gson.JsonObject
import com.google.gson.JsonParser
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.net.URLEncoder
import java.nio.charset.StandardCharsets

class ApiException(message: String) : Exception(message)

data class ApiConfig(val baseUrl: String)

class NoFrillzApi(
    private val config: ApiConfig,
    private val sessionStore: SessionStore,
    private val client: OkHttpClient = OkHttpClient(),
    private val gson: Gson = Gson(),
) {
    fun login(email: String, password: String): AuthResult {
        val payload = mapOf("email" to email, "password" to password)
        val body = request(path = "/sessions", method = "POST", body = gson.toJson(payload), needsAuth = false)
        return decodeAuthResult(body)
    }

    fun signup(
        firstName: String,
        lastName: String,
        username: String,
        email: String,
        password: String,
        about: String?,
    ): AuthResult {
        val payload = linkedMapOf<String, Any>(
            "first_name" to firstName,
            "last_name" to lastName,
            "username" to username,
            "email" to email,
            "password" to password,
        )
        if (!about.isNullOrBlank()) payload["about"] = about

        request(path = "/users", method = "POST", body = gson.toJson(payload), needsAuth = false)
        return login(email = email, password = password)
    }

    fun fetchFeedPage(limit: Int = 25, cursor: String? = null): CursorPage<Post> {
        val query = linkedMapOf("limit" to limit.toString())
        if (!cursor.isNullOrBlank()) query["cursor"] = cursor
        val body = request(path = "/feed", method = "GET", query = query, needsAuth = true)
        return decodePostsPage(body, listKeys = listOf("posts"))
    }

    fun fetchPost(id: String): Post {
        val body = request(path = "/posts/$id", method = "GET", needsAuth = true)
        return decodePostObject(body)
    }

    fun createPost(content: String): Post {
        val payload = mapOf("body" to content)
        val body = request(path = "/posts", method = "POST", body = gson.toJson(payload), needsAuth = true)
        return decodePostObject(body)
    }

    fun likePost(id: String) {
        request(path = "/posts/$id/like", method = "POST", needsAuth = true)
    }

    fun unlikePost(id: String) {
        request(path = "/posts/$id/like", method = "DELETE", needsAuth = true)
    }

    fun fetchCurrentUser(): User {
        val userId = sessionStore.currentUser?.id
            ?: throw ApiException("Missing current user id")
        val body = request(path = "/users/$userId", method = "GET", needsAuth = true)
        return decodeUserObject(body)
    }

    fun fetchUser(id: String): User {
        val body = request(path = "/users/$id", method = "GET", needsAuth = true)
        return decodeUserObject(body)
    }

    fun fetchPostsPage(userId: String, limit: Int = 20, cursor: String? = null): CursorPage<Post> {
        val query = linkedMapOf("limit" to limit.toString())
        if (!cursor.isNullOrBlank()) query["before_id"] = cursor
        val body = request(path = "/users/$userId/posts", method = "GET", query = query, needsAuth = true)
        return decodePostsPage(body, listKeys = listOf("posts"))
    }

    fun searchUsers(query: String, limit: Int = 20): List<User> {
        val body = request(
            path = "/users/search",
            method = "GET",
            query = linkedMapOf("q" to query, "limit" to limit.toString()),
            needsAuth = true,
        )
        return decodeUsersPage(body, keys = listOf("users")).items
    }

    fun followUser(userId: String) {
        request(path = "/users/$userId/follow", method = "POST", needsAuth = true)
    }

    fun unfollowUser(userId: String) {
        request(path = "/users/$userId/unfollow", method = "POST", needsAuth = true)
    }

    fun fetchFollowersPage(userId: String, limit: Int = 25, cursor: String? = null): CursorPage<User> {
        val query = linkedMapOf("limit" to limit.toString())
        if (!cursor.isNullOrBlank()) query["cursor"] = cursor
        val body = request(path = "/users/$userId/followers", method = "GET", query = query, needsAuth = true)
        return decodeUsersPage(body, keys = listOf("followers", "users"))
    }

    fun fetchFollowingPage(userId: String, limit: Int = 25, cursor: String? = null): CursorPage<User> {
        val query = linkedMapOf("limit" to limit.toString())
        if (!cursor.isNullOrBlank()) query["cursor"] = cursor
        val body = request(path = "/users/$userId/following", method = "GET", query = query, needsAuth = true)

        val decodedUsersPage = runCatching {
            decodeUsersPage(body, keys = listOf("following", "users"))
        }.getOrNull()
        if (decodedUsersPage != null) {
            return decodedUsersPage
        }

        val root = parseJson(body)
        val ids = (root as? JsonArray)?.mapNotNull { it.asStringSafe() }
            ?: root.asJsonObjectOrNull()?.get("data")?.asJsonArray?.mapNotNull { it.asStringSafe() }
            ?: emptyList()

        val users = ids.map { fetchUser(it) }
        val nextCursor = if (ids.size >= limit) ids.lastOrNull() else null
        return CursorPage(users, nextCursor)
    }

    fun signOutCurrentSession() {
        request(path = "/sessions/current", method = "DELETE", needsAuth = true)
    }

    private fun refreshSession() {
        val refreshToken = sessionStore.refreshToken ?: throw ApiException("Unauthorized")
        val payload = gson.toJson(mapOf("refresh_token" to refreshToken))
        val body = request(
            path = "/sessions/refresh",
            method = "POST",
            body = payload,
            needsAuth = false,
            retryOnUnauthorized = false,
        )
        val refreshed = decodeAuthResult(body)
        val persistedRefresh = refreshed.refreshToken ?: refreshToken
        sessionStore.saveSession(
            token = refreshed.token,
            refreshToken = persistedRefresh,
            user = refreshed.user,
            userId = refreshed.userId ?: sessionStore.currentUser?.id,
        )
    }

    private fun request(
        path: String,
        method: String,
        query: Map<String, String> = emptyMap(),
        body: String? = null,
        needsAuth: Boolean,
        retryOnUnauthorized: Boolean = true,
    ): String {
        val urlBuilder = StringBuilder(config.baseUrl.trimEnd('/')).append(path)
        if (query.isNotEmpty()) {
            urlBuilder.append("?")
            urlBuilder.append(query.entries.joinToString("&") { (key, value) ->
                "${urlEncode(key)}=${urlEncode(value)}"
            })
        }

        val requestBuilder = Request.Builder()
            .url(urlBuilder.toString())
            .header("Content-Type", "application/json")

        if (needsAuth) {
            val token = sessionStore.authToken
            if (!token.isNullOrBlank()) {
                requestBuilder.header("Authorization", "Bearer $token")
            }
        }

        val jsonMediaType = "application/json; charset=utf-8".toMediaType()
        when (method) {
            "GET" -> requestBuilder.get()
            "POST" -> requestBuilder.post((body ?: "{}").toRequestBody(jsonMediaType))
            "DELETE" -> {
                if (body != null) {
                    requestBuilder.delete(body.toRequestBody(jsonMediaType))
                } else {
                    requestBuilder.delete()
                }
            }
            else -> throw ApiException("Unsupported HTTP method: $method")
        }

        val response = try {
            client.newCall(requestBuilder.build()).execute()
        } catch (e: IOException) {
            throw ApiException(e.message ?: "Network error")
        }

        response.use {
            val payload = it.body?.string().orEmpty()
            if (it.code in 200..299) return payload

            if (it.code == 401) {
                if (needsAuth && retryOnUnauthorized) {
                    return try {
                        refreshSession()
                        request(
                            path = path,
                            method = method,
                            query = query,
                            body = body,
                            needsAuth = needsAuth,
                            retryOnUnauthorized = false,
                        )
                    } catch (_: Throwable) {
                        sessionStore.clear()
                        throw ApiException("Unauthorized")
                    }
                }
                if (needsAuth) sessionStore.clear()
                throw ApiException("Unauthorized")
            }

            throw ApiException(extractErrorMessage(payload) ?: "Request failed (${it.code})")
        }
    }

    private fun extractErrorMessage(payload: String): String? {
        val root = runCatching { parseJson(payload) }.getOrNull() ?: return null
        val obj = root.asJsonObjectOrNull() ?: return null
        val direct = obj.stringOrNull("message", "error")
        if (!direct.isNullOrBlank()) return direct

        val errors = obj.get("errors")
        if (errors is JsonArray) {
            val first = errors.firstOrNull()?.asStringSafe()
            if (!first.isNullOrBlank()) return first
        }

        val dataObj = obj.get("data").asJsonObjectOrNull()
        return dataObj?.stringOrNull("message", "error")
    }

    private fun decodeAuthResult(payload: String): AuthResult {
        val root = parseJson(payload)

        fun fromObject(obj: JsonObject): AuthResult? {
            val token = obj.stringOrNull(
                "token",
                "accessToken",
                "access_token",
                "session_token",
                "jwt",
            ) ?: return null
            val refresh = obj.stringOrNull("refresh_token", "refreshToken")
            val userObj = obj.get("user").asJsonObjectOrNull()
            val user = userObj?.let { decodeUser(it) }
            val userId = obj.stringOrNull("userId", "user_id") ?: user?.id
            return AuthResult(token = token, refreshToken = refresh, user = user, userId = userId)
        }

        val rootObj = root.asJsonObjectOrNull()
        if (rootObj != null) {
            fromObject(rootObj)?.let { return it }
            val dataObj = rootObj.get("data").asJsonObjectOrNull()
            if (dataObj != null) {
                fromObject(dataObj)?.let { return it }
            }
            val sessionObj = rootObj.get("session").asJsonObjectOrNull()
            if (sessionObj != null) {
                fromObject(sessionObj)?.let { return it }
            }
        }

        throw ApiException("Could not decode auth response")
    }

    private fun decodeUsersPage(payload: String, keys: List<String>): CursorPage<User> {
        val root = parseJson(payload)

        if (root is JsonArray) {
            return CursorPage(root.mapNotNull { it.asJsonObjectOrNull()?.let(::decodeUser) }, null)
        }

        val obj = root.asJsonObjectOrNull() ?: throw ApiException("Invalid users payload")

        val directArray = keys.firstNotNullOfOrNull { key -> obj.get(key) as? JsonArray }
        if (directArray != null) {
            return CursorPage(
                items = directArray.mapNotNull { it.asJsonObjectOrNull()?.let(::decodeUser) },
                nextCursor = obj.stringOrNull("next_cursor", "nextCursor", "cursor"),
            )
        }

        val dataElement = obj.get("data")
        if (dataElement is JsonArray) {
            return CursorPage(
                items = dataElement.mapNotNull { it.asJsonObjectOrNull()?.let(::decodeUser) },
                nextCursor = obj.stringOrNull("next_cursor", "nextCursor", "cursor"),
            )
        }
        val dataObj = dataElement.asJsonObjectOrNull()
        if (dataObj != null) {
            val nestedArray = keys.firstNotNullOfOrNull { key -> dataObj.get(key) as? JsonArray }
            if (nestedArray != null) {
                return CursorPage(
                    items = nestedArray.mapNotNull { it.asJsonObjectOrNull()?.let(::decodeUser) },
                    nextCursor = dataObj.stringOrNull("next_cursor", "nextCursor", "cursor"),
                )
            }
        }

        throw ApiException("Could not decode users")
    }

    private fun decodePostsPage(payload: String, listKeys: List<String>): CursorPage<Post> {
        val root = parseJson(payload)

        if (root is JsonArray) {
            return CursorPage(root.mapNotNull { it.asJsonObjectOrNull()?.let(::decodePost) }, null)
        }

        val obj = root.asJsonObjectOrNull() ?: throw ApiException("Invalid posts payload")

        val directArray = listKeys.firstNotNullOfOrNull { key -> obj.get(key) as? JsonArray }
        if (directArray != null) {
            return CursorPage(
                items = directArray.mapNotNull { it.asJsonObjectOrNull()?.let(::decodePost) },
                nextCursor = obj.stringOrNull("next_cursor", "nextCursor", "cursor"),
            )
        }

        val dataElement = obj.get("data")
        if (dataElement is JsonArray) {
            return CursorPage(
                items = dataElement.mapNotNull { it.asJsonObjectOrNull()?.let(::decodePost) },
                nextCursor = obj.stringOrNull("next_cursor", "nextCursor", "cursor"),
            )
        }

        val dataObj = dataElement.asJsonObjectOrNull()
        if (dataObj != null) {
            val nestedArray = listKeys.firstNotNullOfOrNull { key -> dataObj.get(key) as? JsonArray }
            if (nestedArray != null) {
                return CursorPage(
                    items = nestedArray.mapNotNull { it.asJsonObjectOrNull()?.let(::decodePost) },
                    nextCursor = dataObj.stringOrNull("next_cursor", "nextCursor", "cursor"),
                )
            }
        }

        throw ApiException("Could not decode posts")
    }

    private fun decodeUserObject(payload: String): User {
        val root = parseJson(payload)
        root.asJsonObjectOrNull()?.let { obj ->
            if (obj.has("data")) {
                obj.get("data").asJsonObjectOrNull()?.let { return decodeUser(it) }
            }
            if (obj.has("user")) {
                obj.get("user").asJsonObjectOrNull()?.let { return decodeUser(it) }
            }
            return decodeUser(obj)
        }
        throw ApiException("Could not decode user")
    }

    private fun decodePostObject(payload: String): Post {
        val root = parseJson(payload)
        root.asJsonObjectOrNull()?.let { obj ->
            if (obj.has("data")) {
                obj.get("data").asJsonObjectOrNull()?.let { return decodePost(it) }
            }
            if (obj.has("post")) {
                obj.get("post").asJsonObjectOrNull()?.let { return decodePost(it) }
            }
            return decodePost(obj)
        }
        throw ApiException("Could not decode post")
    }

    private fun decodeUser(obj: JsonObject): User {
        val id = obj.stringOrNull("id", "userId", "user_id")
            ?: throw ApiException("Missing user id")
        val username = obj.stringOrNull("username", "handle") ?: "anonymous"
        val joined = obj.stringOrNull("joinedAt", "createdAt")?.toInstantOrNull()

        return User(
            id = id,
            username = username,
            firstName = obj.stringOrNull("first_name", "firstName"),
            lastName = obj.stringOrNull("last_name", "lastName"),
            email = obj.stringOrNull("email"),
            about = obj.stringOrNull("about"),
            accountType = obj.stringOrNull("account_type", "accountType") ?: "human",
            avatarUrl = obj.stringOrNull("avatar_url", "avatarURL", "avatarUrl"),
            joinedAt = joined,
            isFollowing = obj.boolOrDefault(false, "is_following", "isFollowing"),
            isFollower = obj.boolOrDefault(false, "is_follower", "isFollower"),
            followerCount = obj.intOrDefault(0, "follower_count", "followerCount"),
            followingCount = obj.intOrDefault(0, "following_count", "followingCount"),
            postCount = obj.intOrDefault(0, "post_count", "postCount"),
        )
    }

    private fun decodePost(obj: JsonObject): Post {
        val id = obj.stringOrNull("id", "postId", "post_id")
            ?: throw ApiException("Missing post id")

        val userObj = obj.get("user").asJsonObjectOrNull()
            ?: throw ApiException("Missing post user")

        val authorId = userObj.stringOrNull("id", "user_id", "userId")
            ?: throw ApiException("Missing post user id")
        val authorUsername = userObj.stringOrNull("username") ?: "anonymous"

        return Post(
            id = id,
            authorId = authorId,
            authorUsername = authorUsername,
            authorFirstName = userObj.stringOrNull("first_name", "firstName"),
            authorLastName = userObj.stringOrNull("last_name", "lastName"),
            authorAvatarUrl = userObj.stringOrNull("avatar_url", "avatarURL", "avatarUrl"),
            content = obj.stringOrNull("body", "content").orEmpty(),
            createdAt = obj.stringOrNull("createdAt", "created")?.toInstantOrNull(),
            source = obj.stringOrNull("source") ?: "human",
            liked = obj.boolOrDefault(false, "liked", "is_liked"),
            likeCount = obj.intOrDefault(0, "like_count", "likeCount"),
            commentCount = obj.intOrDefault(0, "comment_count", "commentCount"),
        )
    }

    private fun parseJson(payload: String): JsonElement {
        return JsonParser.parseString(payload.ifBlank { "{}" })
    }

    private fun urlEncode(value: String): String {
        return URLEncoder.encode(value, StandardCharsets.UTF_8.name())
    }
}
