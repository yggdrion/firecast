import requests
import json
import os
from pathlib import Path


def load_env():
    env_file = Path(".env")
    if env_file.exists():
        with open(env_file, "r") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    key, value = line.split("=", 1)
                    os.environ[key.strip()] = value.strip()


load_env()

API_KEY = os.getenv("FIRECAST_SECRET", "supersecret")

headers = {"X-API-KEY": API_KEY, "Content-Type": "application/json"}


def make_request(method, endpoint, data=None):
    """Make a simple HTTP request"""
    url = f"http://localhost:8000{endpoint}"
    try:
        response = None
        if method == "GET":
            response = requests.get(url, headers=headers)
        elif method == "POST":
            response = requests.post(url, headers=headers, json=data)

        if response:
            print(f"{method} {url} -> {response.status_code}")
            print(json.dumps(response.json(), indent=2))
            print("-" * 50)
        return response
    except Exception as e:
        print(f"Error: {e}")
        return None



# Health check
make_request("GET", "/healthz")

# Test endpoint
make_request("GET", "/test")

# Get playlists
make_request("GET", "/playlists")

# Add video (comment out if not needed)
# video_data = {
#     "video_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
#     "playlist_id": "6"
# }
# make_request("POST", "/addvideo", video_data)

# Add another video example
# video_data = {
#     "video_url": "https://www.youtube.com/watch?v=LDU_Txk06tM",
#     "playlist_id": "6"
# }
# make_request("POST", "/addvideo", video_data)
