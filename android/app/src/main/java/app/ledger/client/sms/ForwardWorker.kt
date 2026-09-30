package app.ledger.client.sms

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import app.ledger.client.capture.CaptureLog
import app.ledger.client.net.ApiClient
import app.ledger.client.net.DeviceCredentials
import app.ledger.client.net.IngestRequest

/**
 * Forwards one allow-listed SMS to the backend over HTTPS (FR-1.3).
 * WorkManager + CONNECTED constraint gives the offline queue + retry;
 * the server treats ingestion as idempotent via client_message_id.
 */
class ForwardWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        val sender = inputData.getString(KEY_SENDER) ?: return Result.failure()
        val body = inputData.getString(KEY_BODY) ?: return Result.failure()
        val receivedAt = inputData.getLong(KEY_RECEIVED_AT, 0L)
        val clientMessageId = inputData.getString(KEY_CLIENT_MESSAGE_ID) ?: return Result.failure()

        val creds = DeviceCredentials.load(applicationContext)
        if (creds == null) {
            // Not paired yet: retry later — pairing happens during onboarding.
            return Result.retry()
        }

        return try {
            val resp = ApiClient.instance(creds.baseUrl).ingestSms(
                authorization = "Bearer ${creds.accessToken}",
                deviceKey = creds.deviceApiKey,
                request = IngestRequest(
                    sender_id = sender,
                    body = body,
                    received_at = java.time.Instant.ofEpochMilli(receivedAt).toString(),
                    client_message_id = clientMessageId,
                ),
            )
            CaptureLog.markForwarded(applicationContext, clientMessageId)
            when (resp.code()) {
                200, 201, 202 -> Result.success()
                // 409 duplicate: already ingested — success by idempotency.
                409 -> Result.success()
                // 401: token expired → try refresh once, else re-login prompt.
                401 -> {
                    if (creds.refresh(applicationContext)) Result.retry() else Result.failure()
                }
                // 422 permanent payload problem: do not hammer the server.
                422 -> Result.failure()
                else -> Result.retry()
            }
        } catch (e: java.io.IOException) {
            // Offline: WorkManager retries with exponential backoff.
            Result.retry()
        }
    }

    companion object {
        const val KEY_SENDER = "sender"
        const val KEY_BODY = "body"
        const val KEY_RECEIVED_AT = "received_at_ms"
        const val KEY_CLIENT_MESSAGE_ID = "client_message_id"
    }
}
