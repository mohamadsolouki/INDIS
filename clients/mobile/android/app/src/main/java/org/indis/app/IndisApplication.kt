package org.indis.app

import android.app.Application
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import org.indis.app.service.RevocationCacheWorker

class IndisApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        // RTL-first default locale baseline.
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.forLanguageTags("fa"))
        // Schedule the 6-hour revocation list background sync (PRD FR-006).
        RevocationCacheWorker.schedule(this)
    }
}
