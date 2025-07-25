document.addEventListener("DOMContentLoaded", async function () {
  const form = document.getElementById("settingsForm");
  const urlInput = document.getElementById("firecastUrl");
  const secretInput = document.getElementById("firecastSecret");
  const testBtn = document.getElementById("testBtn");
  const resetBtn = document.getElementById("resetBtn");
  const status = document.getElementById("status");

  // Load saved settings
  loadSettings();

  // Save settings
  form.addEventListener("submit", async function (e) {
    e.preventDefault();
    await saveSettings();
  });

  // Test connection
  testBtn.addEventListener("click", async function () {
    await testConnection();
  });

  // Reset settings
  resetBtn.addEventListener("click", async function () {
    await resetSettings();
  });

  async function loadSettings() {
    try {
      const settings = await browser.storage.sync.get([
        "firecastUrl",
        "firecastSecret",
      ]);

      if (settings.firecastUrl) {
        urlInput.value = settings.firecastUrl;
      }

      if (settings.firecastSecret) {
        secretInput.value = settings.firecastSecret;
      }
    } catch (error) {
      showStatus("Failed to load settings: " + error.message, "error");
    }
  }

  async function saveSettings() {
    try {
      const url = urlInput.value.trim();
      const secret = secretInput.value.trim();

      if (!url) {
        showStatus("Please enter a Firecast URL", "error");
        return;
      }

      // Validate URL format
      try {
        new URL(url);
      } catch {
        showStatus("Please enter a valid URL", "error");
        return;
      }

      await browser.storage.sync.set({
        firecastUrl: url,
        firecastSecret: secret,
      });

      showStatus("Settings saved successfully!", "success");
    } catch (error) {
      showStatus("Failed to save settings: " + error.message, "error");
    }
  }

  async function testConnection() {
    try {
      const url = urlInput.value.trim();
      const secret = secretInput.value.trim();

      if (!url) {
        showStatus("Please enter a Firecast URL first", "error");
        return;
      }

      showStatus("Testing connection...", "info");

      const testUrl = url.endsWith("/")
        ? url + "playlists"
        : url + "/playlists";
      const headers = {};

      if (secret) {
        headers["Authorization"] = "Bearer " + secret;
      }

      const response = await fetch(testUrl, {
        headers,
        method: "GET",
      });

      if (response.ok) {
        const data = await response.json();
        const playlistCount = Object.keys(data).length;
        showStatus(
          `Connection successful! Found ${playlistCount} playlists.`,
          "success"
        );
      } else {
        showStatus(
          `Connection failed: HTTP ${response.status} ${response.statusText}`,
          "error"
        );
      }
    } catch (error) {
      showStatus("Connection test failed: " + error.message, "error");
    }
  }

  async function resetSettings() {
    try {
      await browser.storage.sync.clear();
      urlInput.value = "";
      secretInput.value = "";
      showStatus("Settings reset successfully!", "success");
    } catch (error) {
      showStatus("Failed to reset settings: " + error.message, "error");
    }
  }

  function showStatus(message, type) {
    status.textContent = message;
    status.className = "status " + type;
    status.style.display = "block";

    // Auto-hide success messages
    if (type === "success") {
      setTimeout(() => {
        status.style.display = "none";
      }, 3000);
    }
  }
});
