package app.ledger.client

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import app.ledger.client.capture.CaptureLog
import app.ledger.client.net.ApiClient
import app.ledger.client.net.DeviceCredentials
import app.ledger.client.net.LoginRequest
import app.ledger.client.net.PairRequest
import app.ledger.client.sms.AllowList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Minimal single-screen client (SRS §4.1): permission rationale + grant,
 * login/pairing, allow-list view, capture log for transparency.
 */
class MainActivity : AppCompatActivity() {

    private val ui by lazy {
        LinearLayout(this).apply { 
            orientation = LinearLayout.VERTICAL
            setPadding(48, 96, 48, 48)
            setBackgroundColor(android.graphics.Color.parseColor("#000000"))
        }
    }

    private fun getBentoBackground() = android.graphics.drawable.GradientDrawable().apply {
        setColor(android.graphics.Color.parseColor("#050505"))
        setStroke(2, android.graphics.Color.parseColor("#262626"))
        cornerRadius = 32f
    }
    
    private fun getButtonBackground() = android.graphics.drawable.GradientDrawable().apply {
        setColor(android.graphics.Color.parseColor("#ededed"))
        cornerRadius = 16f
    }
    
    private fun getInputBackground() = android.graphics.drawable.GradientDrawable().apply {
        setColor(android.graphics.Color.parseColor("#000000"))
        setStroke(2, android.graphics.Color.parseColor("#404040"))
        cornerRadius = 16f
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (hasSmsPermissions()) {
            renderMain()
        } else {
            renderRationale()
        }
        setContentView(ui)
    }

    // FR-1.1 / US-6: explicit plain-language rationale BEFORE the OS prompt.
    private fun renderRationale() {
        ui.removeAllViews()
        val title = TextView(this).apply {
            text = getString(R.string.rationale_title)
            textSize = 24f
            setTextColor(android.graphics.Color.parseColor("#ffffff"))
            setTypeface(null, android.graphics.Typeface.BOLD)
        }
        val body = TextView(this).apply {
            text = getString(R.string.rationale_body)
            textSize = 15f
            setPadding(0, 32, 0, 64)
            setTextColor(android.graphics.Color.parseColor("#a3a3a3"))
        }
        val grant = Button(this).apply {
            text = getString(R.string.rationale_button)
            setTextColor(android.graphics.Color.parseColor("#000000"))
            background = getButtonBackground()
            setOnClickListener { requestSmsPermissions() }
        }
        ui.addView(title); ui.addView(body); ui.addView(grant)
    }

    private fun hasSmsPermissions() =
        ContextCompat.checkSelfPermission(this, Manifest.permission.RECEIVE_SMS) == PackageManager.PERMISSION_GRANTED &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.READ_SMS) == PackageManager.PERMISSION_GRANTED

    private fun requestSmsPermissions() {
        ActivityCompat.requestPermissions(
            this,
            arrayOf(Manifest.permission.RECEIVE_SMS, Manifest.permission.READ_SMS),
            REQ_SMS,
        )
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == REQ_SMS) {
            if (grantResults.isNotEmpty() && grantResults.all { it == PackageManager.PERMISSION_GRANTED }) {
                renderMain()
            } else {
                Toast.makeText(this, R.string.permission_denied, Toast.LENGTH_LONG).show()
                renderRationale()
            }
        }
    }

    private fun renderMain() {
        ui.removeAllViews()

        val status = TextView(this).apply {
            val isPaired = DeviceCredentials.load(this@MainActivity) != null
            text = if (isPaired) getString(R.string.status_paired) else getString(R.string.status_not_paired)
            textSize = 14f
            setTextColor(android.graphics.Color.parseColor(if (isPaired) "#10b981" else "#ef4444"))
            setPadding(0, 0, 0, 48)
            setTypeface(null, android.graphics.Typeface.BOLD)
        }

        val emailField = android.widget.EditText(this).apply {
            hint = getString(R.string.hint_email)
            setHintTextColor(android.graphics.Color.parseColor("#737373"))
            setTextColor(android.graphics.Color.parseColor("#ffffff"))
            background = getInputBackground()
            setPadding(40, 32, 40, 32)
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(0, 0, 0, 24)
            }
        }
        val passField = android.widget.EditText(this).apply {
            hint = getString(R.string.hint_password)
            setHintTextColor(android.graphics.Color.parseColor("#737373"))
            setTextColor(android.graphics.Color.parseColor("#ffffff"))
            background = getInputBackground()
            setPadding(40, 32, 40, 32)
            transformationMethod = android.text.method.PasswordTransformationMethod.getInstance()
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(0, 0, 0, 48)
            }
        }
        val loginBtn = Button(this).apply {
            text = getString(R.string.button_login)
            setTextColor(android.graphics.Color.parseColor("#000000"))
            background = getButtonBackground()
            setOnClickListener {
                CoroutineScope(Dispatchers.Main).launch { loginAndPair(emailField.text.toString(), passField.text.toString()) }
            }
        }

        val bentoBox = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            background = getBentoBackground()
            setPadding(48, 48, 48, 48)
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(0, 64, 0, 0)
            }
        }

        val allowTitle = TextView(this).apply {
            text = getString(R.string.allowlist_title)
            setPadding(0, 0, 0, 16)
            textSize = 16f
            setTextColor(android.graphics.Color.parseColor("#ffffff"))
            setTypeface(null, android.graphics.Typeface.BOLD)
        }
        val allowList = TextView(this).apply {
            text = AllowList.current(this@MainActivity).joinToString("\n") { "• $it" }
            textSize = 14f
            setTextColor(android.graphics.Color.parseColor("#a3a3a3"))
        }
        val logTitle = TextView(this).apply {
            text = getString(R.string.capture_log_title)
            setPadding(0, 64, 0, 16)
            textSize = 16f
            setTextColor(android.graphics.Color.parseColor("#ffffff"))
            setTypeface(null, android.graphics.Typeface.BOLD)
        }
        val logView = TextView(this).apply {
            text = CaptureLog.entries(this@MainActivity).takeLast(10).joinToString("\n\n")
            textSize = 12f
            setTextColor(android.graphics.Color.parseColor("#737373"))
        }

        bentoBox.addView(allowTitle); bentoBox.addView(allowList)
        bentoBox.addView(logTitle); bentoBox.addView(logView)

        ui.addView(status)
        if (DeviceCredentials.load(this@MainActivity) == null) {
            ui.addView(emailField); ui.addView(passField); ui.addView(loginBtn)
        }
        ui.addView(bentoBox)
    }

    private suspend fun loginAndPair(email: String, password: String) {
        withContext(Dispatchers.IO) {
            try {
                val api = ApiClient.instance(DeviceCredentials.defaultBaseUrl())
                val login = api.login(LoginRequest(email, password))
                if (!login.isSuccessful) {
                    withContext(Dispatchers.Main) { Toast.makeText(this@MainActivity, R.string.login_failed, Toast.LENGTH_LONG).show() }
                    return@withContext
                }
                val tokens = login.body()!!
                val pair = api.pair(
                    "Bearer ${tokens.access_token}",
                    PairRequest(android.os.Build.MODEL, android.os.Build.VERSION.RELEASE),
                )
                if (!pair.isSuccessful) {
                    withContext(Dispatchers.Main) { Toast.makeText(this@MainActivity, R.string.pair_failed, Toast.LENGTH_LONG).show() }
                    return@withContext
                }
                DeviceCredentials.store(
                    this@MainActivity,
                    DeviceCredentials.defaultBaseUrl(),
                    tokens.access_token, tokens.refresh_token,
                    pair.body()!!.device_api_key, email, password,
                )
                withContext(Dispatchers.Main) {
                    Toast.makeText(this@MainActivity, R.string.pair_ok, Toast.LENGTH_SHORT).show()
                    renderMain()
                }
            } catch (e: java.io.IOException) {
                withContext(Dispatchers.Main) { Toast.makeText(this@MainActivity, R.string.network_error, Toast.LENGTH_LONG).show() }
            }
        }
    }

    companion object {
        private const val REQ_SMS = 41
    }
}
