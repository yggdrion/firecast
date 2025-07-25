# Firecast Firefox Extension - Installation Guide

## Quick Start

Your Firefox extension is ready! Here's how to install and use it:

### 1. Load the Extension in Firefox

1. Open Firefox
2. Type `about:debugging` in the address bar and press Enter
3. Click on "This Firefox" in the left sidebar
4. Click "Load Temporary Add-on..."
5. Navigate to this folder and select `manifest.json`
6. The Firecast extension should now appear in your extensions list

### 2. Configure the Extension

1. Click the Firecast icon in your Firefox toolbar (orange play button)
2. Click the settings icon (⚙️) in the popup
3. Enter your Firecast server URL (e.g., `http://localhost:8080` or your server URL)
4. Optionally enter your API secret if required
5. Click "Test Connection" to verify it works
6. Click "Save Settings"

### 3. Use the Extension

1. Navigate to any webpage (YouTube, etc.)
2. Click the Firecast extension icon
3. You'll see a list of your playlists
4. Click on any playlist to add the current page URL to that playlist

## Extension Features

✅ **Playlist Management**: Browse and select from your existing playlists  
✅ **One-Click Addition**: Add videos to playlists with a single click  
✅ **Settings Management**: Configure server URL and authentication  
✅ **Connection Testing**: Verify your server connection  
✅ **Clean UI**: Modern, responsive interface  
✅ **Error Handling**: Clear error messages and status updates

## API Endpoints Used

- `GET /playlists` - Fetches available playlists
- `POST /video/add` - Adds video to selected playlist

## Troubleshooting

**Extension not loading?**

- Make sure all files are in the same directory
- Check the browser console for errors (F12)

**Can't connect to server?**

- Verify your server URL is correct and accessible
- Check if your server requires authentication
- Ensure CORS is properly configured on your server

**Playlists not showing?**

- Check the response format from `/playlists` endpoint
- Expected format: `{"playlist_name": playlist_id, ...}`

## Development Notes

The extension uses:

- Manifest V2 (compatible with current Firefox versions)
- WebExtensions API for storage and tabs
- Modern JavaScript (async/await)
- Clean, accessible HTML/CSS

## Next Steps

You can enhance the extension by:

- Adding proper PNG icons (replace the SVG data URLs)
- Adding video thumbnail previews
- Implementing playlist creation
- Adding bulk video operations
- Improving error handling and retry logic
