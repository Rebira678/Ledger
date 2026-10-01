package app.ledger.client.sms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import androidx.work.BackoffPolicy
import androidx.work.Data
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import app.ledger.client.capture.CaptureLog
import java.util.concurrent.TimeUnit

/**
 * SMS broadcast receiver (FR-1).
 *
 * Runs the MINIMUM logic before handing off to a background worker:
 *  1. read incoming SMS pdus,
 *  2. check sender against the local allow-list (non-matching content is
 *     discarded here and never persisted or transmitted — FR-1.4),
 *  3. append to the in-app capture log for transparency (FR-1.5),
 *  4. enqueue a WorkManager job that performs the HTTPS forwarding with
 *     offline queueing + retry (FR-1.3).
 *
 * No network I/O happens on this path.
 */
class SmsReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) return

        val msgs = Telephony.Sms.Intents.getMessagesFromIntent(intent) ?: return
        if (msgs.isEmpty()) return
        
        val sender = msgs[0].originatingAddress ?: return
        val body = msgs.joinToString("") { it.messageBody ?: "" }
        val receivedAtMs = msgs[0].timestampMillis

        if (!AllowList.matches(context, sender)) {
            // Non-financial SMS: discard on-device, never log content, never send.
            CaptureLog.append(context, sender, allowed = false, forwarded = false)
            return
        }

        // Allowed: persist to the capture log (metadata only) and enqueue.
        CaptureLog.append(context, sender, allowed = true, forwarded = false)
        enqueueForward(context, sender, body, receivedAtMs)
    }

    private fun enqueueForward(context: Context, sender: String, body: String, receivedAtMs: Long) {
        val data = Data.Builder()
            .putString(ForwardWorker.KEY_SENDER, sender)
            .putString(ForwardWorker.KEY_BODY, body)
            .putLong(ForwardWorker.KEY_RECEIVED_AT, receivedAtMs)
            .putString(ForwardWorker.KEY_CLIENT_MESSAGE_ID, ClientMessageIds.newId())
            .build()

        // Unique work per message: retried sends REUSE the same client_message_id,
        // which the server de-duplicates (idempotent ingestion, FR-2.3).
        val request = OneTimeWorkRequestBuilder<ForwardWorker>()
            .setInputData(data)
            .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 30, TimeUnit.SECONDS)
            .setConstraints(
                androidx.work.Constraints.Builder()
                    .setRequiredNetworkType(androidx.work.NetworkType.CONNECTED)
                    .build()
            )
            .build()

        WorkManager.getInstance(context).enqueueUniqueWork(
            "forward-${data.getString(ForwardWorker.KEY_CLIENT_MESSAGE_ID)}",
            androidx.work.ExistingWorkPolicy.KEEP,
            request,
        )
    }
}
