# Real bank/telecom SMS samples for parser validation
#
# HOW TO FILL THIS IN:
# 1. Forward or copy the SMS EXACTLY as it appears on your phone (structure
#    and punctuation matter more than the actual values).
# 2. Redact sensitive data — keep the FORMAT, mask the values:
#      - Account numbers:  1000123456789  ->  1000****6789
#      - Phone numbers:    0911234567     ->  0911***567
#      - Names:            ALMAZ TESFAYE  ->  ALMAZ T.  (optional)
#      - Balances/amounts can stay real or be tweaked, your choice.
# 3. One message per block, with its sender ID as shown in your SMS app.
# 4. Save the file and tell me "samples are ready" — I will parse this file,
#    compare against the parsers, and fix any mismatched formats.
#
# Types of messages worth collecting (2-3 of each is ideal):
#   [ ] CBE credit   (money received)
#   [ ] CBE debit    (payment / transfer out)
#   [ ] CBE other    (balance inquiry, deposit, withdrawal, airtime)
#   [ ] Telebirr received
#   [ ] Telebirr paid / sent
#   [ ] Any OTHER bank/telecom you use (Awash, Dashen, M-Pesa, ...)
#
# =====================================================================
# COPY YOUR MESSAGES BELOW THIS LINE
# =====================================================================

--- SAMPLE 1 ---
sender_id:
date_received:
body:
(replace this line with the exact SMS text)

--- SAMPLE 2 ---
sender_id:
date_received:
body:

--- SAMPLE 3 ---
sender_id:
date_received:
body:

# =====================================================================
# END OF SAMPLES
# =====================================================================
