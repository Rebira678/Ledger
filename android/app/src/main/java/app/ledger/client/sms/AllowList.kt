package app.ledger.client.sms

import android.content.Context

/**
 * Local allow-list of bank/telecom sender IDs (FR-1.2).
 * Default list ships with the app; updated entries persist in shared prefs.
 * Non-matching messages are discarded on-device and NEVER transmitted (FR-1.4).
 */
object AllowList {

    // Default bank/telecom sender IDs for Ethiopia (PRD scope: CBE, Telebirr +).
    // sender IDs are case-insensitive; spaces/hyphens stripped before compare.
    private val DEFAULTS = setOf(
        "CBE", "CBE-BIRR", "127",                 // Commercial Bank of Ethiopia
        "TELEBIRR", "ETHIO-TELECOM", "127-99",    // Telebirr / Ethio Telecom
        "AWASH", "DASHEN",                        // additional banks (parser roadmap)
    )

    private const val PREFS = "ledger_allowlist"
    private const val KEY_EXTRA = "extra_senders"
    private const val KEY_REMOVED = "removed_senders"

    private fun prefs(ctx: Context) =
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun current(ctx: Context): Set<String> {
        val extra = prefs(ctx).getStringSet(KEY_EXTRA, emptySet()) ?: emptySet()
        val removed = prefs(ctx).getStringSet(KEY_REMOVED, emptySet()) ?: emptySet()
        return (DEFAULTS + extra) - removed
    }

    fun add(ctx: Context, senderId: String) {
        val norm = normalize(senderId)
        if (norm.isEmpty()) return
        prefs(ctx).edit().apply {
            putStringSet(KEY_EXTRA, (prefs(ctx).getStringSet(KEY_EXTRA, emptySet()) ?: emptySet()) + norm)
            putStringSet(KEY_REMOVED, (prefs(ctx).getStringSet(KEY_REMOVED, emptySet()) ?: emptySet()) - norm)
            apply()
        }
    }

    fun remove(ctx: Context, senderId: String) {
        val norm = normalize(senderId)
        if (norm.isEmpty()) return
        prefs(ctx).edit().apply {
            putStringSet(KEY_REMOVED, (prefs(ctx).getStringSet(KEY_REMOVED, emptySet()) ?: emptySet()) + norm)
            putStringSet(KEY_EXTRA, (prefs(ctx).getStringSet(KEY_EXTRA, emptySet()) ?: emptySet()) - norm)
            apply()
        }
    }

    /** True when the sender matches the allow-list — the ONLY gate for forwarding. */
    fun matches(ctx: Context, senderId: String): Boolean {
        val norm = normalize(senderId)
        if (norm.isEmpty()) return false
        return current(ctx).any { normalize(it) == norm }
    }

    private fun normalize(s: String): String =
        s.trim().uppercase().replace(" ", "").replace("-", "")
}
