package com.rossotechnologies.nofrillz.core

import com.google.gson.JsonElement
import com.google.gson.JsonObject
import java.time.Instant
import java.time.format.DateTimeParseException

data class User(
    val id: String,
    val username: String,
    val firstName: String? = null,
    val lastName: String? = null,
    val email: String? = null,
    val about: String? = null,
    val accountType: String = "human",
    val avatarUrl: String? = null,
    val joinedAt: Instant? = null,
    val isFollowing: Boolean = false,
    val isFollower: Boolean = false,
    val followerCount: Int = 0,
    val followingCount: Int = 0,
    val postCount: Int = 0,
) {
    val fullName: String?
        get() {
            val first = firstName.orEmpty().trim()
            val last = lastName.orEmpty().trim()
            val combined = "$first $last".trim()
            return combined.ifEmpty { null }
        }

    val displayName: String
        get() = fullName ?: username

    val initials: String
        get() {
            val first = firstName.orEmpty().trim().take(1)
            val last = lastName.orEmpty().trim().take(1)
            val combined = "$first$last".lowercase()
            return if (combined.isBlank()) username.take(1).lowercase() else combined
        }
}

data class Post(
    val id: String,
    val authorId: String,
    val authorUsername: String,
    val authorFirstName: String? = null,
    val authorLastName: String? = null,
    val authorAvatarUrl: String? = null,
    val content: String,
    val createdAt: Instant? = null,
    val source: String = "human",
    val liked: Boolean = false,
    val likeCount: Int = 0,
    val commentCount: Int = 0,
) {
    val authorDisplayName: String
        get() {
            val first = authorFirstName.orEmpty().trim()
            val last = authorLastName.orEmpty().trim()
            val combined = "$first $last".trim()
            return if (combined.isNotEmpty()) combined else authorUsername
        }

    val isAi: Boolean
        get() = source.equals("ai", ignoreCase = true)

    fun withLiked(nextLiked: Boolean): Post {
        val updatedLikeCount = when {
            nextLiked == liked -> likeCount
            nextLiked -> likeCount + 1
            else -> (likeCount - 1).coerceAtLeast(0)
        }
        return copy(liked = nextLiked, likeCount = updatedLikeCount)
    }
}

data class AuthResult(
    val token: String,
    val refreshToken: String?,
    val user: User?,
    val userId: String?,
)

data class CursorPage<T>(
    val items: List<T>,
    val nextCursor: String?,
)

internal fun JsonObject.stringOrNull(vararg keys: String): String? {
    for (key in keys) {
        val value = get(key) ?: continue
        val asString = value.asStringSafe() ?: continue
        if (asString.isNotBlank()) return asString
    }
    return null
}

internal fun JsonObject.intOrDefault(default: Int = 0, vararg keys: String): Int {
    for (key in keys) {
        val value = get(key) ?: continue
        val parsed = value.asIntSafe() ?: continue
        return parsed
    }
    return default
}

internal fun JsonObject.boolOrDefault(default: Boolean = false, vararg keys: String): Boolean {
    for (key in keys) {
        val value = get(key) ?: continue
        val parsed = value.asBooleanSafe() ?: continue
        return parsed
    }
    return default
}

internal fun JsonElement?.asJsonObjectOrNull(): JsonObject? {
    if (this == null || !isJsonObject) return null
    return asJsonObject
}

internal fun JsonElement.asStringSafe(): String? {
    return try {
        when {
            isJsonNull -> null
            isJsonPrimitive -> asJsonPrimitive.asString
            else -> null
        }
    } catch (_: Throwable) {
        null
    }
}

internal fun JsonElement.asIntSafe(): Int? {
    return try {
        when {
            isJsonPrimitive && asJsonPrimitive.isNumber -> asInt
            isJsonPrimitive && asJsonPrimitive.isString -> asString.toIntOrNull()
            else -> null
        }
    } catch (_: Throwable) {
        null
    }
}

internal fun JsonElement.asBooleanSafe(): Boolean? {
    return try {
        when {
            isJsonPrimitive && asJsonPrimitive.isBoolean -> asBoolean
            isJsonPrimitive && asJsonPrimitive.isString -> asString.toBooleanStrictOrNull()
            else -> null
        }
    } catch (_: Throwable) {
        null
    }
}

internal fun String.toInstantOrNull(): Instant? {
    return try {
        Instant.parse(this)
    } catch (_: DateTimeParseException) {
        null
    }
}
