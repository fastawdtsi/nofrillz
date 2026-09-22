package com.rossotechnologies.nofrillz.core

import android.content.Context
import com.rossotechnologies.nofrillz.BuildConfig

class AppContainer(context: Context) {
    val sessionStore = SessionStore(context)
    val api = NoFrillzApi(
        config = ApiConfig(baseUrl = BuildConfig.NOFRILLZ_API_BASE_URL),
        sessionStore = sessionStore,
    )
}
