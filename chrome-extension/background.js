let authToken = "";
let selectedRepo = "";

// Listen for messages from popup
chrome.runtime.onMessage.addListener((msg) => {
  if (msg.type === "SET_TOKEN") {
    authToken = msg.token;
  }

  if (msg.type === "SET_REPO") {
    selectedRepo = msg.repo;
  }
});

// Auto-sync every 2 minutes
setInterval(() => {
  if (!authToken || !selectedRepo) return;

  const [owner, repo] = selectedRepo.split("/");

  fetch(`http://localhost:8080/sync?owner=${owner}&repo=${repo}`, {
    headers: {
      Authorization: authToken
    }
  })
    .then(res => res.text())
    .then(msg => {
      showNotification("Repo Updated", msg);
    })
    .catch(() => {});
}, 120000); // 2 minutes

function showNotification(title, message) {
  chrome.notifications.create({
    type: "basic",
    iconUrl: "icon.png",
    title: title,
    message: message
  });
}
