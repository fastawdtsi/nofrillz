package com.rossotechnologies.nofrillz.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val DarkColorScheme = darkColorScheme(
    primary = NightInk,
    onPrimary = NightPaper,
    secondary = QuietGray,
    tertiary = LikedRed,
    background = NightPaper,
    onBackground = NightInk,
    surface = NightPaper,
    onSurface = NightInk,
    surfaceVariant = RuleDark,
    onSurfaceVariant = Color(0xFFB8B8B8),
    outline = RuleDark,
)

private val LightColorScheme = lightColorScheme(
    primary = Ink,
    onPrimary = Paper,
    secondary = QuietGray,
    tertiary = LikedRed,
    background = Paper,
    onBackground = Ink,
    surface = Paper,
    onSurface = Ink,
    surfaceVariant = RuleLight,
    onSurfaceVariant = QuietGray,
    outline = RuleLight,
)

@Composable
fun NoFrillzTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content
    )
}
