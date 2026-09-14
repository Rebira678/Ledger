# Ledger Android Client

Kotlin client (minSdk 26 / Android 8.0) that captures bank/telecom SMS, filters
by a local allow-list, and forwards qualifying messages to the Ledger backend
with offline queueing via WorkManager.

## Feature mapping (SRS FR-1)

| ID | Requirement | Where |
|---|---|---|
| FR-1.1 | Plain-language rationale before OS prompt | `MainActivity.renderRationale()` |
| FR-1.2 | Local updatable sender-ID allow-list | `sms/AllowList.kt` |
| FR-1.3 | HTTPS forwarding + offline queue/retry | `sms/ForwardWorker.kt` (WorkManager, exponential backoff) |
| FR-1.4 | Never persist/transmit non-matching SMS | `sms/SmsReceiver.kt` discards before any logging |
| FR-1.5 | In-app capture log | `capture/CaptureLog.kt` (metadata only, no bodies) |

Idempotency: every message gets a client-generated `client_message_id`
(`sms/ClientMessageIds.kt`) reused across retries; the backend de-duplicates.

## Building

```bash
cd android
./gradlew assembleDebug          # debug APK for sideload testing
./gradlew assembleRelease        # debug-signed release APK (see below)
```

## Sideload distribution (PRD §7)

1. `./gradlew assembleRelease` → `app/build/outputs/apk/release/app-release.apk`
2. Host the APK on the landing page with these instructions:
   - Settings → Security → **Install unknown apps** → allow your browser/file manager
   - Open the APK → Install → confirm the Play-Protect warning (expected for sideloaded apps; Ledger is not distributed via Play yet)
3. Release signing requires a real keystore (see `DECISIONS.md` Known Gaps);
   v1 ships debug-signed for portfolio/testing distribution.

## What needs a real device (cannot be validated in a sandbox)

- Runtime permission prompts and rationale display
- Actual `SMS_RECEIVED` delivery timing and dual-SIM behavior
- Battery-optimization / OEM task-killer interference with background receivers
- Real CBE/Telebirr message samples to confirm end-to-end parsing
