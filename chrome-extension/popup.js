let authToken = "";
let currentChart = null;

// ----------------------------
// INITIALIZE - CHECK FOR SAVED TOKEN
// ----------------------------
document.addEventListener('DOMContentLoaded', () => {
    console.log("🔍 Popup loaded, checking storage...");

    // Check if user is already authenticated
    chrome.storage.local.get(['authToken', 'selectedRepo'], (result) => {
        console.log("📦 Storage result:", result);

        if (result.authToken) {
            console.log("✅ Auth token found");
            authToken = result.authToken;
            hideAuthShowMain();
            showLogoutButton();

            // Load repositories and restore selection after loading
            loadRepositories(result.selectedRepo);
        } else {
            console.log("⚠️ No auth token found");
        }
    });
});

// ----------------------------
// LOGOUT FUNCTIONALITY
// ----------------------------
document.getElementById("logout").addEventListener("click", () => {
    // Clear token and repo from storage and memory
    chrome.storage.local.remove(['authToken', 'selectedRepo'], () => {
        authToken = "";

        // Notify background script to stop auto-sync
        chrome.runtime.sendMessage({ type: "CLEAR" });

        // Reset UI to login state
        document.getElementById("authSection").classList.remove("hidden");
        document.getElementById("mainContent").classList.remove("active");
        document.getElementById("logout").classList.add("hidden");

        // Reset repository select
        document.getElementById("repoSelect").innerHTML = '<option value="">Choose a repository...</option>';

        // Hide chart and stats
        document.getElementById("chartSection").classList.add("hidden");
        document.getElementById("statsSection").classList.add("hidden");

        // Destroy chart if exists
        if (currentChart) {
            currentChart.destroy();
            currentChart = null;
        }

        showStatus("👋 Logged out successfully", "info");
    });
});

function showLogoutButton() {
    document.getElementById("logout").classList.remove("hidden");
}

// ----------------------------
// HELPER FUNCTIONS
// ----------------------------
function showStatus(message, type = 'info') {
    const status = document.getElementById("status");
    status.textContent = message;
    status.className = `status-message ${type}`;

    // Auto-hide success messages after 3 seconds
    if (type === 'success') {
        setTimeout(() => {
            status.textContent = '';
            status.className = 'status-message';
        }, 3000);
    }
}

function showLoading(message) {
    const status = document.getElementById("status");
    status.innerHTML = `<span class="spinner"></span>${message}`;
    status.className = 'status-message loading';
}

function hideAuthShowMain() {
    document.getElementById("authSection").classList.add("hidden");
    document.getElementById("mainContent").classList.add("active");
}

function enableSyncButton() {
    const syncBtn = document.getElementById("sync");
    syncBtn.disabled = false;
}

// ----------------------------
// LOGIN WITH GITHUB
// ----------------------------
document.getElementById("login").addEventListener("click", () => {
    showLoading("Opening GitHub login...");

    // Use window.open() instead of chrome.tabs.create() so postMessage works
    window.open(
        "http://localhost:8080/auth/github",
        "_blank",
        "width=600,height=700"
    );
});

// ----------------------------
// RECEIVE TOKEN FROM BACKEND
// ----------------------------
window.addEventListener("message", (event) => {
    if (event.data.token) {
        authToken = event.data.token;

        // Save token to Chrome storage for persistence
        chrome.storage.local.set({ authToken: authToken }, () => {
            // Send token to background script for auto-sync
            chrome.runtime.sendMessage({ type: "SET_TOKEN", token: authToken });

            showStatus("✅ Logged in successfully", "success");
            hideAuthShowMain();
            showLogoutButton();
            loadRepositories();
        });
    }
});

// ----------------------------
// LOAD USER REPOSITORIES
// ----------------------------
function loadRepositories(selectedRepo = null) {
    showLoading("Loading repositories...");

    // fetch("https://gitsense-ooly.onrender.com/repos", {
    fetch("http://localhost:8080/repos", {
        headers: {
            "Authorization": authToken
        }
    })
        .then(res => res.json())
        .then(repos => {
            const select = document.getElementById("repoSelect");
            select.innerHTML = '<option value="">Choose a repository...</option>';

            repos.forEach(repo => {
                const option = document.createElement("option");
                option.value = `${repo.owner.login}/${repo.name}`;
                option.textContent = `${repo.owner.login}/${repo.name}`;
                select.appendChild(option);
            });

            showStatus(`✅ Loaded ${repos.length} repositories`, "success");

            // Restore selected repo after repositories are loaded
            if (selectedRepo) {
                console.log("📂 Restoring selected repo:", selectedRepo);
                select.value = selectedRepo;
                console.log("🔧 Set select value to:", select.value);

                if (select.value) {
                    document.getElementById("sync").disabled = false;

                    // Check if repo has been synced before and load the chart
                    const repoName = selectedRepo.split('/')[1];
                    console.log("🔍 Checking for existing data for repo:", repoName);
                    checkAndLoadExistingData(repoName);
                } else {
                    console.log("❌ Select value is empty after setting");
                }
            }
        })
        .catch(() => {
            showStatus("❌ Failed to load repositories", "error");
        });
}

// ----------------------------
// ENABLE SYNC WHEN REPO SELECTED
// ----------------------------
document.getElementById("repoSelect").addEventListener("change", (e) => {
    const syncBtn = document.getElementById("sync");
    const repoValue = e.target.value;

    syncBtn.disabled = !repoValue;

    if (repoValue) {
        // Save selected repo to storage
        chrome.storage.local.set({ selectedRepo: repoValue }, () => {
            // Send repo to background script for auto-sync
            chrome.runtime.sendMessage({ type: "SET_REPO", repo: repoValue });
        });
    }
});

// ----------------------------
// SYNC REPO + SAVE SNAPSHOT
// ----------------------------
document.getElementById("sync").addEventListener("click", () => {
    const repoValue = document.getElementById("repoSelect").value;

    if (!repoValue) {
        showStatus("⚠️ Please select a repository", "error");
        return;
    }

    const [owner, repo] = repoValue.split("/");

    showLoading("Syncing repository...");

    // fetch(`https://gitsense-ooly.onrender.com/sync?owner=${owner}&repo=${repo}`, {
    fetch(`http://localhost:8080/sync?owner=${owner}&repo=${repo}`, {
        headers: {
            "Authorization": authToken
        }
    })
        .then(res => res.text())
        .then(msg => {
            if (msg.includes("🔔")) {
                showStatus("🔔 Significant activity detected!", "info");

                chrome.notifications.create({
                    type: "basic",
                    iconUrl: "icon.png",
                    title: "GitSense Alert",
                    message: "Significant repo activity detected!"
                });
            } else {
                showStatus("✅ Sync completed successfully", "success");
            }

            loadHistory(repo);
            loadCommits(repo);
            updateLastSyncTime();

        })
        .catch(() => {
            showStatus("❌ Sync failed - please try again", "error");
        });
});

// ----------------------------
// VIEW FULL DASHBOARD
// ----------------------------
document.getElementById("viewDashboard").addEventListener("click", () => {
    const repoValue = document.getElementById("repoSelect").value;

    if (!repoValue) {
        showStatus("⚠️ Please select a repository", "error");
        return;
    }

    // Open dashboard in new tab with repo parameter
    const dashboardUrl = `http://localhost:8080/dashboard?repo=${encodeURIComponent(repoValue)}`;
    chrome.tabs.create({ url: dashboardUrl });
});

// ----------------------------
// UPDATE LAST SYNC TIME
// ----------------------------
function updateLastSyncTime() {
    const lastSyncElement = document.getElementById("lastSync");
    const now = new Date();
    const timeString = now.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit'
    });
    lastSyncElement.textContent = timeString;

    // Show stats section
    document.getElementById("statsSection").classList.remove("hidden");
}

// ----------------------------
// CHECK IF REPO HAS EXISTING DATA
// ----------------------------
function checkAndLoadExistingData(repo) {
    console.log("📊 Fetching history for repo:", repo);

    // Silently check if repo has been synced before
    fetch(`http://localhost:8080/history?repo=${repo}`)
        .then(res => {
            console.log("📡 Response status:", res.status);
            return res.json();
        })
        .then(data => {
            console.log("📊 History data received:", data);
            console.log("📊 Data length:", data ? data.length : 0);

            if (data && data.length > 0) {
                console.log("✅ Repo has been synced before - loading chart");

                // Repo has been synced before - render the chart with existing data
                renderChart(data);

                // Load commits list
                loadCommits(repo);

                // Show the last sync time from the most recent snapshot
                const lastSnapshot = data[data.length - 1];
                const lastSyncElement = document.getElementById("lastSync");
                const syncTime = new Date(lastSnapshot.time);
                lastSyncElement.textContent = syncTime.toLocaleTimeString('en-US', {
                    hour: '2-digit',
                    minute: '2-digit'
                });

                // Show stats section
                document.getElementById("statsSection").classList.remove("hidden");
                console.log("✅ Chart loaded successfully");
            } else {
                console.log("⚠️ No data found - user needs to sync");
            }
            // If no data, user will need to click sync button (expected behavior)
        })
        .catch((err) => {
            console.error("❌ Error checking existing data:", err);
            // Silently fail - user will just need to sync manually
        });
}

// ----------------------------
// RENDER CHART WITH DATA
// ----------------------------
function renderChart(data) {
    // Destroy previous chart if exists
    if (currentChart) {
        currentChart.destroy();
    }

    const ctx = document.getElementById("chart").getContext("2d");
    const labels = data.map(d => {
        const date = new Date(d.time);
        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    });
    const scores = data.map(d => d.score);

    // Calculate average score
    const avgScore = scores.reduce((a, b) => a + b, 0) / scores.length;
    document.getElementById("activityScore").textContent = Math.round(avgScore);

    currentChart = new Chart(ctx, {
        type: "line",
        data: {
            labels: labels,
            datasets: [
                {
                    label: "Activity Score",
                    data: scores,
                    borderColor: "#667eea",
                    backgroundColor: "rgba(102, 126, 234, 0.15)",
                    fill: true,
                    tension: 0.4,
                    borderWidth: 3,
                    pointRadius: 4,
                    pointHoverRadius: 6,
                    pointBackgroundColor: "#667eea",
                    pointBorderColor: "#fff",
                    pointBorderWidth: 2,
                    pointHoverBackgroundColor: "#764ba2",
                    pointHoverBorderColor: "#fff"
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    backgroundColor: 'rgba(0, 0, 0, 0.8)',
                    padding: 12,
                    cornerRadius: 8,
                    displayColors: false,
                    callbacks: {
                        label: function (context) {
                            return `Score: ${context.parsed.y}`;
                        }
                    }
                }
            },
            scales: {
                y: {
                    min: 0,
                    max: 100,
                    grid: {
                        color: 'rgba(0, 0, 0, 0.05)',
                        drawBorder: false
                    },
                    ticks: {
                        color: '#718096',
                        font: {
                            size: 11
                        }
                    }
                },
                x: {
                    grid: {
                        display: false,
                        drawBorder: false
                    },
                    ticks: {
                        color: '#718096',
                        font: {
                            size: 10
                        },
                        maxRotation: 0
                    }
                }
            }
        }
    });

    // Show chart section
    document.getElementById("chartSection").classList.remove("hidden");
    document.getElementById("statsSection").classList.remove("hidden");
}

// ----------------------------
// LOAD REPO HISTORY (GRAPH)
// ----------------------------
function loadHistory(repo) {
    showLoading("Loading activity chart...");

    // fetch(`https://gitsense-ooly.onrender.com/history?repo=${repo}`)
    fetch(`http://localhost:8080/history?repo=${repo}`)
        .then(res => res.json())
        .then(data => {
            renderChart(data);
            showStatus("📊 Chart loaded successfully", "success");
        })
        .catch(() => {
            showStatus("❌ Failed to load activity history", "error");
        });
}
function loadCommits(repo) {
    fetch(`http://localhost:8080/commits?repo=${repo}`)
      .then(res => res.json())
      .then(commits => {
        const list = document.getElementById("commitList");
        const section = document.getElementById("commitsSection");

        list.innerHTML = "";

        if (!commits || commits.length === 0) {
          list.innerHTML = "<li>No commits found</li>";
          section.classList.remove("hidden");
          return;
        }

        // Limit to 5 most recent commits
        const recentCommits = commits.slice(0, 5);

        recentCommits.forEach(c => {
          const li = document.createElement("li");
          li.textContent = `${c.date.split("T")[0]} – ${c.author}: ${c.message}`;
          list.appendChild(li);
        });

        // Show commits section
        section.classList.remove("hidden");

        // Show "View Dashboard" button
        document.getElementById("viewDashboard").style.display = "block";
      })
      .catch(() => {
        console.error("❌ Failed to load commits");
      });
  }

