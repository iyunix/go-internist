import requests

# DIRECTLY provide your API key here
API_KEY = "aa-hmGFFZQ31CbqHvh4Ce4Zg7xbb584k3eNirvlqMppILA2wbqZ"

# Example Persian text for translation
persian_text = "ماکسیمم دوز هیدرورلازین"

# API URL for Avalai's OpenAI-compatible chat completion endpoint
url = "https://api.avalai.ir/v1/chat/completions"

# Payload in OpenAI-style chat format
payload = {
    "model": "gemini-2.5-flash-lite",  # or the model you want
    "messages": [
        {
            "role": "system",
            "content": "Translate all user input Persian medical text to clear, precise English. Return only English, nothing else."
        },
        {
            "role": "user",
            "content": persian_text
        }
    ]
}
headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json",
}

response = requests.post(url, json=payload, headers=headers)

print("Status:", response.status_code)
print("Body:", response.text)

if response.ok:
    data = response.json()
    # Extract translation from choices[0].message.content
    try:
        translation = data['choices'][0]['message']['content']
        print("TRANSLATED:", translation)
    except Exception as e:
        print("ERROR extracting translation:", e)
else:
    print("FAIL!")
