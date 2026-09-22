package com.rossotechnologies.nofrillz.core

private val emailRegex = Regex("^[A-Za-z0-9+_.-]+@[A-Za-z0-9.-]+$")
private val usernameRegex = Regex("^[A-Za-z0-9_]{3,32}$")

object AuthInputValidator {
    fun isValidEmail(email: String): Boolean = emailRegex.matches(email)

    fun isValidUsername(username: String): Boolean = usernameRegex.matches(username)
}
