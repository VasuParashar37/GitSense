// Background script for auto-syncing repository activity

// ----------------------------
// MESSAGE LISTENER FROM POPUP
// ----------------------------
chrome.runtime.onMessage.addListener((msg) => {
  if (msg.type === "SET_TOKEN") {
    chrome.storage.local.set({ authToken: msg.token });
    console.log("✅ Token saved for background sync");
  }

  if (msg.type === "SET_REPO") {
    chrome.storage.local.set({ selectedRepo: msg.repo });
    console.log("✅ Repository set for background sync:", msg.repo);
  }

  if (msg.type === "CLEAR") {
    chrome.storage.local.remove(['authToken', 'selectedRepo']);
    console.log("✅ Background sync cleared");
  }
});

// ----------------------------
// AUTO-SYNC FUNCTION
// ----------------------------
async function autoSync() {
  // Get token and repo from storage
  const result = await chrome.storage.local.get(['authToken', 'selectedRepo']);

  const { authToken, selectedRepo } = result;

  // Exit if not authenticated or no repo selected
  if (!authToken || !selectedRepo) {
    return;
  }

  const [owner, repo] = selectedRepo.split("/");

  console.log(`🔄 Auto-syncing ${selectedRepo}...`);

  try {
    // Sync the repository
    const response = await fetch(`http://localhost:8080/sync?owner=${owner}&repo=${repo}`, {
      headers: {
        Authorization: authToken
      }
    });

    const msg = await response.text();

    // Only notify if significant activity detected
    if (msg.includes("🔔")) {
      console.log("🔔 Significant activity detected!");

      chrome.notifications.create({
        type: "basic",
        iconUrl: "icon.png",
        title: "GitSense Alert",
        message: `Significant activity detected in ${repo}!`
      });
    } else {
      console.log("✅ Auto-sync completed (no significant activity)");
    }
  } catch (error) {
    console.error("❌ Auto-sync failed:", error);
  }
}

// ----------------------------
// RUN AUTO-SYNC EVERY 5 MINUTES
// ----------------------------
setInterval(autoSync, 300000); // 300000 ms = 5 minutes

// Run sync immediately when background script starts (for testing)
autoSync();

console.log("🚀 GitSense background sync initialized (interval: 5 minutes)");
