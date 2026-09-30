import os
import json
import base64
import requests

api_key = os.environ.get("LEDGER_LLM_API_KEY")
base_url = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"

with open("scratch/latest_upload.png", "rb") as f:
    img_b64 = base64.b64encode(f.read()).decode('utf-8')

resp = requests.post(
    base_url,
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}"
    },
    json={
        "model": "gemini-3.6-flash",
        "messages": [{
            "role": "user",
            "content": [
                {"type": "text", "text": "Extract total amount, return JSON: {\"amount\": 123.45}"},
                {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{img_b64}"}}
            ]
        }],
        "temperature": 0.1
    },
    timeout=60
)
print(f"Status: {resp.status_code}")
if resp.status_code != 200:
    print(resp.text[:500])
else:
    print(resp.json()['choices'][0]['message']['content'])
