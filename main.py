from fastapi import (
    FastAPI,
    HTTPException,
    status,
    Request,
)
from pydantic_settings import BaseSettings
from pydantic import ValidationError
import os
import yt_dlp
import time
import logging
import requests
import base64

app = FastAPI()

logging.basicConfig(
    level=logging.INFO,
    format="%(levelname)s %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)


class Settings(BaseSettings):
    FIRECAST_SECRET: str
    AZURACAST_API_KEY: str
    AZURACAST_DOMAIN: str  # Fixed typo here

    class Config:
        env_file = ".env"
        extra = "ignore"


try:
    settings = Settings()  # type: ignore
except ValidationError as e:
    missing = []
    for err in e.errors():
        if err["type"] == "missing":
            missing.append(err["loc"][0])
    if missing:
        raise RuntimeError(f"Missing required environment variables: {', '.join(missing)}") from None
    else:
        raise


def downloadVideoWithYtDlpAsMp3(video_url: str) -> str:
    ydl_opts = {
        "format": "bestaudio/best",
        "postprocessors": [
            {
                "key": "FFmpegExtractAudio",
                "preferredcodec": "mp3",
                "preferredquality": "192",
            }
        ],
        "outtmpl": "%(title)s.%(ext)s",
    }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        info = ydl.extract_info(video_url, download=False)
        filename = ydl.prepare_filename(info)
        mp3_filename = os.path.splitext(filename)[0] + ".mp3"
        ydl.download([video_url])
    return mp3_filename


def upload_to_azuracast(local_file: str):
    api_url = f"https://{settings.AZURACAST_DOMAIN}/api/station/1/files"

    headers = {
        "X-API-Key": settings.AZURACAST_API_KEY,
        "Content-Type": "application/json",
    }

    with open(local_file, "rb") as f:
        file_content = base64.b64encode(f.read()).decode("utf-8")

    data = {"path": os.path.basename(local_file), "file": file_content}

    response = requests.post(api_url, headers=headers, json=data)

    if not response.ok:
        # os.remove(local_file)
        raise Exception(f"AzuraCast API upload error: {response.status_code} {response.text}")

    print(f"Uploaded {local_file} to AzuraCast")
    os.remove(local_file)


def add_song_to_azuracast_playlist(filename: str, playlist: str):
    api_url = f"https://{settings.AZURACAST_DOMAIN}/api/station/1/playlist/{playlist}/import"

    headers = {
        "X-API-Key": settings.AZURACAST_API_KEY,
        "Content-Type": "application/json",
    }
    data = {"path": filename}
    response = requests.post(api_url, headers=headers, json=data)
    if not response.ok:
        raise Exception(f"AzuraCast API error: {response.status_code} {response.text}")


@app.middleware("http")
async def log_requests(request: Request, call_next):
    start_time = time.time()
    client_ip = request.client.host if request.client else "unknown"
    method = request.method
    path = request.url.path
    response = await call_next(request)
    status_code = response.status_code
    process_time = (time.time() - start_time) * 1000  # ms
    timestamp = time.strftime("%Y-%m-%d %H:%M:%S", time.localtime(start_time))
    # Color by status code
    if 200 <= status_code < 300:
        color = "\033[92m"  # Green
    elif 300 <= status_code < 400:
        color = "\033[94m"  # Blue
    elif 400 <= status_code < 500:
        color = "\033[93m"  # Yellow
    else:
        color = "\033[91m"  # Red
    reset = "\033[0m"
    log_msg = f"{timestamp} {client_ip} {method} {path} {color}{status_code}{reset} {process_time:.2f}ms"
    logging.info(log_msg)
    return response


@app.get("/")
@app.get("/healthz")
def root():
    return {"status": "ok"}


@app.get("/test")
def test():
    api_url = f"https://{settings.AZURACAST_DOMAIN}/api/station/1/files"

    # Noisestorm - Crab Rave [Monstercat Release].mp3

    headers = {
        "X-API-Key": settings.AZURACAST_API_KEY,
        "Content-Type": "application/json",
    }
    response = requests.get(api_url, headers=headers)
    if not response.ok:
        raise Exception(f"AzuraCast API error: {response.status_code} {response.text}")

    try:
        data = response.json()
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to parse JSON response: {str(e)}",
        )
    if not isinstance(data, list):
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Unexpected response format from AzuraCast API",
        )

    return data


@app.get("/playlists")
def playlists():
    api_url = f"https://{settings.AZURACAST_DOMAIN}/api/station/1/playlists"

    # Noisestorm - Crab Rave [Monstercat Release].mp3

    headers = {
        "X-API-Key": settings.AZURACAST_API_KEY,
        "Content-Type": "application/json",
    }
    response = requests.get(api_url, headers=headers)
    if not response.ok:
        raise Exception(f"AzuraCast API error: {response.status_code} {response.text}")

    try:
        data = response.json()
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to parse JSON response: {str(e)}",
        )
    if not isinstance(data, list):
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Unexpected response format from AzuraCast API",
        )

    playlists = {playlist["name"]: playlist["id"] for playlist in data}
    if not playlists:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="No playlists found in AzuraCast",
        )

    return playlists


@app.post("/addvideo")
async def add_video(request: Request):
    api_key = request.headers.get("x-api-key")
    if api_key != settings.FIRECAST_SECRET:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or missing API key",
        )

    body = await request.json()

    video_url = body.get("video_url")
    if not video_url:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Missing 'video_url' in request body",
        )
    playlist = body.get("playlist")
    if not playlist:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Missing 'playlist' in request body",
        )

    print(f"Received video URL: {video_url}, Playlist: {playlist}")

    try:
        mp3_file = downloadVideoWithYtDlpAsMp3(video_url)
        upload_to_azuracast(mp3_file)
        # add_song_to_azuracast_playlist(os.path.basename(mp3_file), playlist)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error downloading, uploading, or adding to playlist: {str(e)}",
        )

    return {"message": f"File '{mp3_file}' processed and uploaded successfully."}
