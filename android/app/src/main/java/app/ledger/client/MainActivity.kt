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
            setPadding(64, 128, 64, 64)
            setBackgroundColor(android.graphics.Color.parseColor("#050505"))
        }
    }

    private fun getBentoBackground() = android.graphics.drawable.GradientDrawable().apply {
        setColor(android.graphics.Color.parseColor("#0f0f11"))
        setStroke(3, android.graphics.Color.parseColor("#27272a"))
        cornerRadius = 64f
    }
    
    private fun getButtonBackground() = android.graphics.drawable.GradientDrawable().apply {
        setColor(android.graphics.Color.parseColor("#ededed"))
        cornerRadius = 64f
    }
    
    private fun getInputBackground() = android.graphics.drawable.GradientDrawable().apply {
        setColor(android.graphics.Color.parseColor("#0a0a0a"))
        setStroke(2, android.graphics.Color.parseColor("#262626"))
        cornerRadius = 32f
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        supportActionBar?.hide()
        renderWebView()
        setContentView(ui)
    }

    inner class WebAppInterface {
        @android.webkit.JavascriptInterface
        fun onLogin(email: String, pass: String, token: String) {
            val isPaired = DeviceCredentials.load(this@MainActivity) != null
            if (!isPaired) {
                CoroutineScope(Dispatchers.Main).launch {
                    pairDeviceBackground(email, pass, token)
                }
            } else if (!hasSmsPermissions()) {
                requestSmsPermissions()
            }
        }
    }

    private suspend fun pairDeviceBackground(email: String, pass: String, token: String) {
        withContext(Dispatchers.IO) {
            try {
                val api = ApiClient.instance(DeviceCredentials.defaultBaseUrl())
                val pair = api.pair(
                    "Bearer $token",
                    PairRequest(android.os.Build.MODEL, android.os.Build.VERSION.RELEASE)
                )
                if (pair.isSuccessful) {
                    val pairBody = pair.body()!!
                    DeviceCredentials.store(
                        this@MainActivity,
                        DeviceCredentials.defaultBaseUrl(),
                        token, "dummy_refresh", // web app handles refresh
                        pairBody.device_api_key, email, pass
                    )
                    withContext(Dispatchers.Main) {
                        if (!hasSmsPermissions()) {
                            requestSmsPermissions()
                        } else {
                            showWebNotification("Your phone is now syncing receipts!", false)
                        }
                    }
                }
            } catch (e: Exception) {
                // ignore in UI, agent pairing failed
            }
        }
    }

    private fun renderWebView() {
        ui.removeAllViews()
        ui.setPadding(0, 0, 0, 0)
        
        val webView = android.webkit.WebView(this).apply {
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT, 
                LinearLayout.LayoutParams.MATCH_PARENT
            )
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            settings.mixedContentMode = android.webkit.WebSettings.MIXED_CONTENT_ALWAYS_ALLOW
            
            addJavascriptInterface(WebAppInterface(), "Android")
            
            webViewClient = object : android.webkit.WebViewClient() {
                override fun onPageFinished(view: android.webkit.WebView, url: String) {
                    super.onPageFinished(view, url)
                    val creds = app.ledger.client.net.DeviceCredentials.load(this@MainActivity)
                    val injectTokenJs = if (creds != null) "localStorage.setItem('token', '${creds.accessToken}');" else ""
                    
                    val js = """
                        if (!window.androidInjected) {
                            window.androidInjected = true;
                            
                            // Inject stored token from Android if present
                            if ("$injectTokenJs" !== "") {
                                $injectTokenJs
                                if (window.location.pathname === '/login' || window.location.pathname === '/') {
                                    window.location.href = '/dashboard';
                                }
                            }
                            
                            const originalFetch = window.fetch;
                            window.fetch = async function(...args) {
                                const response = await originalFetch.apply(this, args);
                                if (args[0] && (args[0].includes('/v1/auth/login') || args[0].includes('/v1/auth/register')) && response.ok) {
                                    try {
                                        const clone = response.clone();
                                        const data = await clone.json();
                                        const reqBody = JSON.parse(args[1].body);
                                        Android.onLogin(reqBody.email, reqBody.password, data.access_token);
                                    } catch(e) {}
                                }
                                return response;
                            };
                            
                            var existing = localStorage.getItem('token');
                            if (existing) {
                                Android.onLogin('existing@user', 'dummy', existing);
                            }
                        }
                    """.trimIndent()
                    view.evaluateJavascript(js, null)
                }
            }
        }
        ui.addView(webView)
        val creds = app.ledger.client.net.DeviceCredentials.load(this)
        val base = DeviceCredentials.defaultBaseUrl()
        val url = if (creds != null) {
            if (base.endsWith("/")) "${base}dashboard" else "$base/dashboard"
        } else {
            base
        }
        webView.loadUrl(url)
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
        // If they just granted permissions, we can just show a toast or notification over the WebView!
        if (requestCode == REQ_SMS) {
            if (grantResults.isNotEmpty() && grantResults.all { it == PackageManager.PERMISSION_GRANTED }) {
                Toast.makeText(this, "Your phone is now securely connected to Ledger!", Toast.LENGTH_SHORT).show()
            } else {
                showPermissionRationaleDialog()
            }
        }
    }

    private fun showPermissionRationaleDialog() {
        android.app.AlertDialog.Builder(this)
            .setTitle("Permission Required")
            .setMessage("Ledger needs SMS access to automatically track your bank transactions and update your balance. If you accidentally denied this, please allow it in Settings.")
            .setPositiveButton("Try Again") { _, _ ->
                requestSmsPermissions()
            }
            .setNegativeButton("Open Settings") { _, _ ->
                val intent = android.content.Intent(android.provider.Settings.ACTION_APPLICATION_DETAILS_SETTINGS)
                intent.data = android.net.Uri.fromParts("package", packageName, null)
                startActivity(intent)
            }
            .setCancelable(false)
            .show()
    }

    private fun showWebNotification(message: String, isError: Boolean = false) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show()
    }

    companion object {
        private const val REQ_SMS = 41
    }
}
