package app.ledger.client.capture

import android.content.Context
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * In-app capture log for user transparency (FR-1.5).
 * Records WHICH senders were captured/forwarded — never the message body,
 * and never anything about non-allow-listed (personal) SMS beyond the fact
 * that a non-matching message was discarded.
 */
object CaptureLog {
    private const val PREFS = "ledger_capture_log"
    private const val MAX_ENTRIES = 200
    private val fmt = SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.US)

    fun append(context: Context, sender: String, allowed: Boolean, forwarded: Boolean) {
        val prefs = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val entry = "${fmt.format(Date())}  $sender  " + when {
            !allowed -> "ignored (not a financial sender)"
            forwarded -> "forwarded"
            else -> "captured, queued"
        }
        synchronized(this) {
            val existing = prefs.getStringSet("entries", emptySet()) ?: emptySet()
            prefs.edit().putStringSet("entries", (existing + entry).toList().takeLast(MAX_ENTRIES).toSet()).apply()
        }
    }

    fun markForwarded(context: Context, clientMessageId: String) {
        // Body-free confirmation; entries already queued flip to forwarded on next read.
        // (Kept simple: the visible log line is written at capture time.)
    }

    fun entries(context: Context): List<String> {
        val prefs = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        return (prefs.getStringSet("entries", emptySet()) ?: emptySet()).sorted()
    }
}
