document.addEventListener("DOMContentLoaded", async function () {
  const settingsBtn = document.getElementById("settingsBtn");
  const loading = document.getElementById("loading");
  const error = document.getElementById("error");
  const success = document.getElementById("success");
  const playlistList = document.getElementById("playlistList");

  // Show settings page
  settingsBtn.addEventListener("click", function () {
    browser.runtime.openOptionsPage();
  });

  try {
    // Get current tab URL
    const tabs = await browser.tabs.query({
      active: true,
      currentWindow: true,
    });
    const currentUrl = tabs[0].url;

    // Get settings
    const settings = await browser.storage.sync.get([
      "firecastUrl",
      "firecastSecret",
    ]);

    if (!settings.firecastUrl) {
      showError("Please configure the Firecast URL in settings first.");
      return;
    }

    // Fetch playlists
    const playlists = await fetchPlaylists(
      settings.firecastUrl,
      settings.firecastSecret
    );

    if (playlists) {
      displayPlaylists(playlists, currentUrl, settings);
    }
  } catch (err) {
    showError("Failed to load playlists: " + err.message);
  }
});

async function fetchPlaylists(baseUrl, secret) {
  try {
    loading.style.display = "block";
    error.style.display = "none";

    const url = baseUrl.endsWith("/")
      ? baseUrl + "playlists"
      : baseUrl + "/playlists";
    const headers = {};

    if (secret) {
      headers["Authorization"] = "Bearer " + secret;
    }

    const response = await fetch(url, { headers });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = await response.json();
    loading.style.display = "none";

    return data;
  } catch (err) {
    loading.style.display = "none";
    throw err;
  }
}

function displayPlaylists(playlists, currentUrl, settings) {
  playlistList.innerHTML = "";

  // Color palette for playlists
  const colors = [
    "red",
    "blue",
    "green",
    "purple",
    "orange",
    "pink",
    "teal",
    "yellow",
  ];

  // Convert playlists object to array of [name, id] pairs and sort by name
  const playlistArray = Object.entries(playlists).sort((a, b) =>
    a[0].localeCompare(b[0])
  );

  playlistArray.forEach(([name, id], index) => {
    const listItem = document.createElement("li");

    // Assign color based on index (cycling through colors)
    const colorClass = `color-${colors[index % colors.length]}`;
    listItem.className = `playlist-item ${colorClass}`;

    // Add emoji based on playlist name for extra visual appeal
    const emoji = getPlaylistEmoji(name);
    listItem.innerHTML = `<div class="playlist-name">${emoji} ${escapeHtml(
      name
    )}</div>`;

    listItem.addEventListener("click", async function () {
      try {
        await addVideoToPlaylist(currentUrl, id, settings);
        showSuccess(`Video added to "${name}" playlist successfully!`);

        // Close popup after successful addition
        setTimeout(() => {
          window.close();
        }, 1500);
      } catch (err) {
        showError(`Failed to add video to playlist: ${err.message}`);
      }
    });

    playlistList.appendChild(listItem);
  });

  playlistList.style.display = "block";
}

function getPlaylistEmoji(name) {
  const nameLower = name.toLowerCase();

  // Match common playlist themes with emojis
  if (nameLower.includes("music") || nameLower.includes("song")) return "🎵";
  if (nameLower.includes("movie") || nameLower.includes("film")) return "🎬";
  if (nameLower.includes("comedy") || nameLower.includes("funny")) return "😂";
  if (nameLower.includes("game") || nameLower.includes("gaming")) return "🎮";
  if (nameLower.includes("news") || nameLower.includes("politics")) return "📰";
  if (nameLower.includes("tech") || nameLower.includes("science")) return "🔬";
  if (nameLower.includes("sport") || nameLower.includes("football"))
    return "⚽";
  if (nameLower.includes("food") || nameLower.includes("cooking")) return "🍳";
  if (nameLower.includes("travel") || nameLower.includes("adventure"))
    return "✈️";
  if (nameLower.includes("animal") || nameLower.includes("pet")) return "🐾";
  if (nameLower.includes("art") || nameLower.includes("creative")) return "🎨";
  if (nameLower.includes("education") || nameLower.includes("learn"))
    return "📚";
  if (nameLower.includes("fitness") || nameLower.includes("workout"))
    return "💪";
  if (nameLower.includes("default") || nameLower.includes("main")) return "📋";
  if (nameLower.includes("favorite") || nameLower.includes("fav")) return "⭐";
  if (nameLower.includes("watch later") || nameLower.includes("later"))
    return "⏰";

  // Default emoji
  return "📺";
}

async function addVideoToPlaylist(videoUrl, playlistId, settings) {
  const url = settings.firecastUrl.endsWith("/")
    ? settings.firecastUrl + "video/add"
    : settings.firecastUrl + "/video/add";

  const headers = {
    "Content-Type": "application/json",
  };

  if (settings.firecastSecret) {
    headers["Authorization"] = "Bearer " + settings.firecastSecret;
  }

  const response = await fetch(url, {
    method: "POST",
    headers: headers,
    body: JSON.stringify({
      videoUrl: videoUrl,
      playlistId: playlistId,
    }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`HTTP ${response.status}: ${errorText}`);
  }

  return response.json();
}

function showError(message) {
  loading.style.display = "none";
  success.style.display = "none";
  error.textContent = message;
  error.style.display = "block";
  playlistList.style.display = "none";
}

function showSuccess(message) {
  loading.style.display = "none";
  error.style.display = "none";
  success.textContent = message;
  success.style.display = "block";
  playlistList.style.display = "none";
}

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}
