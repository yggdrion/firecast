# Firecast Firefox Extension

A Firefox extension that allows you to add videos from any webpage to your Firecast playlists.

## Features

- 🎬 Add videos to Firecast playlists with one click
- 📋 Browse and select from available playlists
- ⚙️ Configure your Firecast server URL and authentication
- 🔒 Secure storage of settings
- 🎨 Clean, modern interface

## Installation

1. Open Firefox
2. Navigate to `about:debugging`
3. Click "This Firefox"
4. Click "Load Temporary Add-on"
5. Select the `manifest.json` file from this extension directory

## Setup

1. Click the Firecast extension icon in your toolbar
2. Click the settings icon (⚙️) to open configuration
3. Enter your Firecast server URL (e.g., `https://your-firecast-server.com`)
4. Optionally enter your API secret if your server requires authentication
5. Click "Test Connection" to verify your settings
6. Click "Save Settings"

## Usage

1. Navigate to any webpage with a video
2. Click the Firecast extension icon
3. Select the playlist you want to add the video to
4. The current page URL will be sent to your Firecast server

## API Integration

The extension makes the following API calls:

- `GET /playlists` - Retrieves available playlists
- `POST /video/add` - Adds a video to a playlist

### Request Format for Video Addition

```json
{
  "videoUrl": "https://example.com/video",
  "playlistId": 1
}
```

## Development

The extension consists of:

- `manifest.json` - Extension configuration
- `popup.html/js` - Main popup interface
- `options.html/js` - Settings page
- `icons/` - Extension icons

## Permissions

The extension requires the following permissions:

- `activeTab` - To get the current tab URL
- `storage` - To save settings
- `http://*/*` and `https://*/*` - To make API requests to your Firecast server
