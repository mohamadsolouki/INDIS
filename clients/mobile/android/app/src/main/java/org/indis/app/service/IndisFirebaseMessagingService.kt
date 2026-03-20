package org.indis.app.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import org.indis.app.R
import org.indis.app.ui.home.MainActivity

/**
 * FCM push handler.
 *
 * Notification types dispatched by the INDIS backend (see notification service):
 *  - `credential_issued`   — a new VC has been placed in the citizen's wallet
 *  - `enrollment_update`   — enrollment status changed (approved / rejected)
 *  - `election_reminder`   — upcoming election window opens in <24 h
 *  - `revocation_alert`    — one or more credentials in the wallet were revoked
 */
class IndisFirebaseMessagingService : FirebaseMessagingService() {

    override fun onMessageReceived(message: RemoteMessage) {
        val title = message.notification?.title
            ?: message.data["title"]
            ?: getString(R.string.notif_default_title)
        val body = message.notification?.body
            ?: message.data["body"]
            ?: return  // nothing to show

        showNotification(title, body)
    }

    override fun onNewToken(token: String) {
        // TODO: POST token to gateway /v1/notifications/fcm-token so the
        // notification service can target this device.
        // Implementation deferred until prod Firebase project is configured.
    }

    // ── helpers ──────────────────────────────────────────────────────────────

    private fun showNotification(title: String, body: String) {
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                getString(R.string.notif_channel_name),
                NotificationManager.IMPORTANCE_DEFAULT
            )
            manager.createNotificationChannel(channel)
        }

        val tapIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP
            },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .setContentIntent(tapIntent)
            .build()

        manager.notify(NOTIFICATION_ID++, notification)
    }

    companion object {
        private const val CHANNEL_ID = "indis_main"
        private var NOTIFICATION_ID = 1000
    }
}
