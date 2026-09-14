package app.ledger.client.net

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/**
 * Pairing credentials stored in EncryptedSharedPreferences (data-at-rest
 * protection per SRS §5.2). Access token is short-lived; the refresh token
 * rotates on 401.
 */
data class DeviceCredentials(
    val baseUrl: String,
    val accessToken: String,
    val refreshToken: String,
    val deviceApiKey: String,
) {
    /** Attempt a token refresh; returns true when the access token was replaced. */
    suspend fun refresh(context: Context): Boolean {
        return try {
            val resp = ApiClient.instance(baseUrl).login(
                LoginRequest(storedEmail(context) ?: return false, storedPassword(context) ?: return false)
            )
            if (resp.isSuccessful) {
                val body = resp.body() ?: return false
                store(
                    context, baseUrl, body.access_token, body.refresh_token, deviceApiKey,
                    storedEmail(context) ?: "", storedPassword(context) ?: "",
                )
                true
            } else false
        } catch (e: java.io.IOException) {
            false
        }
    }

    companion object {
        private const val FILE = "ledger_secure_prefs"

        private fun prefs(context: Context) = EncryptedSharedPreferences.create(
            context,
            FILE,
            MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build(),
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )

        fun load(context: Context): DeviceCredentials? {
            val p = prefs(context)
            val base = p.getString("base_url", null) ?: return null
            val access = p.getString("access_token", null) ?: return null
            val refresh = p.getString("refresh_token", null) ?: return null
            val device = p.getString("device_api_key", null) ?: return null
            return DeviceCredentials(base, access, refresh, device)
        }

        fun store(
            context: Context,
            baseUrl: String,
            accessToken: String,
            refreshToken: String,
            deviceApiKey: String,
            email: String,
            password: String,
        ) {
            prefs(context).edit()
                .putString("base_url", baseUrl)
                .putString("access_token", accessToken)
                .putString("refresh_token", refreshToken)
                .putString("device_api_key", deviceApiKey)
                .putString("email", email)
                .putString("password", password)
                .apply()
        }

        fun storedEmail(context: Context): String? = prefs(context).getString("email", null)
        fun storedPassword(context: Context): String? = prefs(context).getString("password", null)

        fun defaultBaseUrl(): String = "https://api.ledger.app/"
    }
}
