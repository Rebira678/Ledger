package app.ledger.client.sms

import java.security.SecureRandom

/** Client-generated unique message IDs for idempotent ingestion (API contract §3). */
object ClientMessageIds {
    private val rnd = SecureRandom()
    private val HEX = "0123456789abcdef".toCharArray()

    /** 16 hex chars — unique per device, generated once per message. */
    fun newId(): String {
        val b = ByteArray(8)
        rnd.nextBytes(b)
        val sb = StringBuilder(16)
        for (x in b) {
            sb.append(HEX[(x.toInt() shr 4) and 0x0f])
            sb.append(HEX[x.toInt() and 0x0f])
        }
        return sb.toString()
    }
}
