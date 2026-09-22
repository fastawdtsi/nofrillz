package com.rossotechnologies.nofrillz.core

import android.content.Context
import android.content.SharedPreferences
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

class SessionStore(context: Context) {
    private val prefs: SharedPreferences = context.getSharedPreferences("nofrillz_session", Context.MODE_PRIVATE)

    private val _sessionState = MutableStateFlow(readSessionState())
    val sessionState: StateFlow<SessionState> = _sessionState.asStateFlow()

    val authToken: String?
        get() = prefs.getString(KEY_AUTH_TOKEN, null)

    val refreshToken: String?
        get() = prefs.getString(KEY_REFRESH_TOKEN, null)

    val currentUser: User?
        get() {
            val id = prefs.getString(KEY_USER_ID, null) ?: return null
            return User(
                id = id,
                username = prefs.getString(KEY_USERNAME, null).orEmpty(),
                firstName = prefs.getString(KEY_FIRST_NAME, null),
                lastName = prefs.getString(KEY_LAST_NAME, null),
                email = prefs.getString(KEY_EMAIL, null),
                about = prefs.getString(KEY_ABOUT, null),
                accountType = prefs.getString(KEY_ACCOUNT_TYPE, null) ?: "human",
                avatarUrl = prefs.getString(KEY_AVATAR_URL, null),
                followerCount = prefs.getInt(KEY_FOLLOWER_COUNT, 0),
                followingCount = prefs.getInt(KEY_FOLLOWING_COUNT, 0),
                postCount = prefs.getInt(KEY_POST_COUNT, 0),
            )
        }

    val isAuthenticated: Boolean
        get() = !authToken.isNullOrBlank()

    fun saveSession(token: String, refreshToken: String?, user: User?, userId: String?) {
        prefs.edit().apply {
            putString(KEY_AUTH_TOKEN, token)
            if (!refreshToken.isNullOrBlank()) {
                putString(KEY_REFRESH_TOKEN, refreshToken)
            } else {
                remove(KEY_REFRESH_TOKEN)
            }
            if (user != null) {
                putString(KEY_USER_ID, user.id)
                putString(KEY_USERNAME, user.username)
                putString(KEY_FIRST_NAME, user.firstName)
                putString(KEY_LAST_NAME, user.lastName)
                putString(KEY_EMAIL, user.email)
                putString(KEY_ABOUT, user.about)
                putString(KEY_ACCOUNT_TYPE, user.accountType)
                putString(KEY_AVATAR_URL, user.avatarUrl)
                putInt(KEY_FOLLOWER_COUNT, user.followerCount)
                putInt(KEY_FOLLOWING_COUNT, user.followingCount)
                putInt(KEY_POST_COUNT, user.postCount)
            } else if (!userId.isNullOrBlank()) {
                putString(KEY_USER_ID, userId)
            }
            apply()
        }
        _sessionState.value = readSessionState()
    }

    fun updateUser(user: User) {
        prefs.edit().apply {
            putString(KEY_USER_ID, user.id)
            putString(KEY_USERNAME, user.username)
            putString(KEY_FIRST_NAME, user.firstName)
            putString(KEY_LAST_NAME, user.lastName)
            putString(KEY_EMAIL, user.email)
            putString(KEY_ABOUT, user.about)
            putString(KEY_ACCOUNT_TYPE, user.accountType)
            putString(KEY_AVATAR_URL, user.avatarUrl)
            putInt(KEY_FOLLOWER_COUNT, user.followerCount)
            putInt(KEY_FOLLOWING_COUNT, user.followingCount)
            putInt(KEY_POST_COUNT, user.postCount)
            apply()
        }
        _sessionState.value = readSessionState()
    }

    fun clear() {
        prefs.edit().clear().apply()
        _sessionState.value = readSessionState()
    }

    private fun readSessionState(): SessionState {
        return SessionState(
            isAuthenticated = isAuthenticated,
            user = currentUser,
        )
    }

    companion object {
        private const val KEY_AUTH_TOKEN = "auth_token"
        private const val KEY_REFRESH_TOKEN = "refresh_token"
        private const val KEY_USER_ID = "user_id"
        private const val KEY_USERNAME = "username"
        private const val KEY_FIRST_NAME = "first_name"
        private const val KEY_LAST_NAME = "last_name"
        private const val KEY_EMAIL = "email"
        private const val KEY_ABOUT = "about"
        private const val KEY_ACCOUNT_TYPE = "account_type"
        private const val KEY_AVATAR_URL = "avatar_url"
        private const val KEY_FOLLOWER_COUNT = "follower_count"
        private const val KEY_FOLLOWING_COUNT = "following_count"
        private const val KEY_POST_COUNT = "post_count"
    }
}

data class SessionState(
    val isAuthenticated: Boolean,
    val user: User?,
)
