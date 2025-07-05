import requests
import os
from dotenv import load_dotenv

load_dotenv()

AZURACAST_API_URL = os.getenv("AZURACAST_API_URL")
AZURACAST_API_KEY = os.getenv("AZURACAST_API_KEY")
AZURACAST_ENDPOINT = os.getenv("AZURACAST_ENDPOINT", "/status")  # Change to any endpoint you want to test

url = AZURACAST_API_URL.rstrip("status") + AZURACAST_ENDPOINT
headers = {"X-API-Key": AZURACAST_API_KEY}

response = requests.get(url, headers=headers)
print(f"Status code: {response.status_code}")
print(f"Response: {response.text}")
