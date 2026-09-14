package app.ledger.client.net

import com.squareup.moshi.KotlinJsonAdapterFactory
import com.squareup.moshi.Moshi
import retrofit2.Response
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import retrofit2.http.Body
import retrofit2.http.Header
import retrofit2.http.POST

/** Request body for POST /v1/ingest/sms (API contract §3). */
data class IngestRequest(
    val sender_id: String,
    val body: String,
    val received_at: String,
    val client_message_id: String,
)

/** Response for POST /v1/ingest/sms. */
data class IngestResponse(
    val transaction_id: String?,
    val status: String,
    val confidence: Double?,
)

/** Request/response for POST /v1/devices/pair (API contract §2). */
data class PairRequest(val device_model: String, val os_version: String)
data class PairResponse(val device_id: String, val device_api_key: String)

/** Request/response for POST /v1/auth/login. */
data class LoginRequest(val email: String, val password: String)
data class LoginResponse(val access_token: String, val refresh_token: String, val expires_in: Long)

interface LedgerApi {
    @POST("v1/ingest/sms")
    suspend fun ingestSms(
        @Header("Authorization") authorization: String,
        @Header("X-Device-Key") deviceKey: String,
        @Body request: IngestRequest,
    ): Response<IngestResponse>

    @POST("v1/devices/pair")
    suspend fun pair(
        @Header("Authorization") authorization: String,
        @Body request: PairRequest,
    ): Response<PairResponse>

    @POST("v1/auth/login")
    suspend fun login(@Body request: LoginRequest): Response<LoginResponse>
}

/** Retrofit client factory (base URL injectable for local dev vs prod). */
object ApiClient {
    private val moshi = Moshi.Builder()
        .add(KotlinJsonAdapterFactory())
        .build()

    fun instance(baseUrl: String): LedgerApi = Retrofit.Builder()
        .baseUrl(if (baseUrl.endsWith("/")) baseUrl else "$baseUrl/")
        .addConverterFactory(MoshiConverterFactory.create(moshi))
        .build()
        .create(LedgerApi::class.java)
}
