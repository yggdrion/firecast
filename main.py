from fastapi import FastAPI, HTTPException, status, Request
from pydantic_settings import BaseSettings
import os
import yt_dlp
import paramiko

app = FastAPI()


class Settings(BaseSettings):
    FIRECAST_SECRET: str
    SFTP_ADDRESS: str
    SFTP_PORT: int
    SFTP_USER: str
    SFTP_PASSWORD: str

    class Config:
        env_file = ".env"


settings = Settings()  # type: ignore


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


def upload_to_sftp(local_file: str):
    transport = paramiko.Transport((settings.SFTP_ADDRESS, settings.SFTP_PORT))
    transport.connect(username=settings.SFTP_USER, password=settings.SFTP_PASSWORD)
    sftp = paramiko.SFTPClient.from_transport(transport)
    if sftp is None:
        transport.close()
        os.remove(local_file)
        raise Exception("Failed to establish SFTP connection.")
    remote_path = os.path.basename(local_file)
    sftp.put(local_file, remote_path)
    sftp.close()
    transport.close()
    print(f"Uploaded {local_file}")
    os.remove(local_file)


@app.get("/")
@app.get("/healthz")
def root():
    return {"status": "ok"}


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

    try:
        mp3_file = downloadVideoWithYtDlpAsMp3(video_url)
        upload_to_sftp(mp3_file)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error downloading or uploading video: {str(e)}",
        )

    return {"message": f"File '{mp3_file}' processed and uploaded successfully."}
