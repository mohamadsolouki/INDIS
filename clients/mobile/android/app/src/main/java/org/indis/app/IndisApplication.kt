package org.indis.app

import android.app.Application
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat

class IndisApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        // RTL-first default locale baseline.
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.forLanguageTags("fa"))
    }
}
