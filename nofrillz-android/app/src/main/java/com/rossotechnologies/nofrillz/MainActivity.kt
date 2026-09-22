package com.rossotechnologies.nofrillz

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import com.rossotechnologies.nofrillz.ui.theme.NoFrillzTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            NoFrillzTheme {
                NoFrillzApp()
            }
        }
    }
}
