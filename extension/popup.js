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

  // Convert playlists object to array of [name, id] pairs and sort by name
  const playlistArray = Object.entries(playlists).sort((a, b) =>
    a[0].localeCompare(b[0])
  );

  playlistArray.forEach(([name, id]) => {
    const listItem = document.createElement("li");
    listItem.className = "playlist-item";
    listItem.innerHTML = `<div class="playlist-name">${escapeHtml(name)}</div>`;

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
