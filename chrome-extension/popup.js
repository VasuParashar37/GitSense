let authToken = "";

// ----------------------------
// LOGIN WITH GITHUB
// ----------------------------
document.getElementById("login").addEventListener("click", () => {
    chrome.tabs.create({
        url: "https://gitsense-ooly.onrender.com/auth/github"
    });
});

// ----------------------------
// RECEIVE TOKEN FROM BACKEND
// ----------------------------
window.addEventListener("message", (event) => {
    if (event.data.token) {
        authToken = event.data.token;

        document.getElementById("status").innerText =
            "✅ Logged in successfully";

        loadRepositories();
    }
});

// ----------------------------
// LOAD USER REPOSITORIES
// ----------------------------
function loadRepositories() {
    fetch("https://gitsense-ooly.onrender.com/repos", {
        headers: {
            "Authorization": authToken
        }
    })
        .then(res => res.json())
        .then(repos => {
            const select = document.getElementById("repoSelect");
            select.innerHTML = "";

            repos.forEach(repo => {
                const option = document.createElement("option");
                option.value = `${repo.owner.login}/${repo.name}`;
                option.textContent = `${repo.owner.login}/${repo.name}`;
                select.appendChild(option);
            });

            document.getElementById("status").innerText =
                "✅ Repositories loaded";
        })
        .catch(() => {
            document.getElementById("status").innerText =
                "❌ Failed to load repositories";
        });
}

// ----------------------------
// SYNC REPO + SAVE SNAPSHOT
// ----------------------------
document.getElementById("sync").addEventListener("click", () => {
    const repoValue = document.getElementById("repoSelect").value;

    if (!repoValue) {
        alert("Please select a repository");
        return;
    }

    const [owner, repo] = repoValue.split("/");

    fetch(`https://gitsense-ooly.onrender.com/sync?owner=${owner}&repo=${repo}`, {
        headers: {
            "Authorization": authToken
        }
    })
        .then(res => res.text())
        .then(msg => {
            document.getElementById("status").innerText = msg;

            if (msg.includes("🔔")) {
                chrome.notifications.create({
                    type: "basic",
                    iconUrl: "icon.png",
                    title: "GitSense Alert",
                    message: "Significant repo activity detected!"
                });
            }

            loadHistory(repoValue);
        })

        .catch(() => {
            document.getElementById("status").innerText =
                "❌ Sync failed";
        });
});

// ----------------------------
// LOAD REPO HISTORY (GRAPH)
// ----------------------------
function loadHistory(repo) {
    fetch(`https://gitsense-ooly.onrender.com/history?repo=${repo}`)
        .then(res => res.json())
        .then(data => {
            const ctx = document.getElementById("chart").getContext("2d");

            const labels = data.map(d => d.time);
            const scores = data.map(d => d.score);

            new Chart(ctx, {
                type: "line",
                data: {
                    labels: labels,
                    datasets: [
                        {
                            label: "Repo Activity",
                            data: scores,
                            borderColor: "blue",
                            backgroundColor: "rgba(0,0,255,0.1)",
                            fill: true,
                            tension: 0.3
                        }
                    ]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            min: 0,
                            max: 100
                        }
                    }
                }
            });
        })
        .catch(() => {
            document.getElementById("status").innerText =
                "❌ Failed to load history";
        });
}
