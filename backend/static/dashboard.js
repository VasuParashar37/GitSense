let allCommits = [];
let filteredCommits = [];
let currentPage = 1;
const commitsPerPage = 20;
let allData = {};
let timelineChart = null;
let commitsPerDayChart = null;
let contributionChart = null;
let latestHistory = [];
let latestFiles = [];
let latestBreakdown = {};

const state = {
  timeRangeDays: 30,
  query: "",
  sortBy: "date",
  sortDir: "desc",
  timelineRange: "7d",
  timelineView: "hourly"
};

const THEME = {
  primary: "#1f7a8c",
  primarySoft: "rgba(31, 122, 140, 0.18)",
  primaryStrong: "#155e6c",
  success: "#2f9a77",
  warning: "#ef8354",
  textMuted: "#4f6d80",
  grid: "rgba(21, 59, 80, 0.12)",
  tooltip: "rgba(12, 38, 54, 0.94)"
};

const urlParams = new URLSearchParams(window.location.search);
const repo = urlParams.get("repo");

if (!repo) {
  document.body.innerHTML = '<div class="empty-state"><div class="icon">⚠️</div><h2>No repository specified</h2><p>Please open the dashboard from the extension.</p></div>';
} else {
  document.getElementById("repoName").textContent = repo;
  document.getElementById("timelineRepoName").textContent = repo;
  setupControls();
  setupTimelineControls();
  loadDashboardData(repo);
}

function setupControls() {
  const rangeEl = document.getElementById("timeRange");
  const searchEl = document.getElementById("commitSearch");
  const refreshEl = document.getElementById("refreshData");

  rangeEl.addEventListener("change", (e) => {
    state.timeRangeDays = Number(e.target.value);
    currentPage = 1;
    applyCommitFiltersAndRender();
  });

  let searchTimer;
  searchEl.addEventListener("input", (e) => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      state.query = e.target.value.trim().toLowerCase();
      currentPage = 1;
      applyCommitFiltersAndRender();
    }, 150);
  });

  refreshEl.addEventListener("click", () => {
    setPageStatus("Refreshing dashboard data...", "info");
    loadDashboardData(repo, true);
  });

  document.querySelectorAll(".sortable").forEach((th) => {
    th.addEventListener("click", () => {
      const key = th.dataset.sort;
      if (state.sortBy === key) {
        state.sortDir = state.sortDir === "asc" ? "desc" : "asc";
      } else {
        state.sortBy = key;
        state.sortDir = key === "date" ? "desc" : "asc";
      }
      currentPage = 1;
      updateSortIndicators();
      applyCommitFiltersAndRender();
    });
  });

  updateSortIndicators();
}

function setupTimelineControls() {
  document.querySelectorAll("#rangeButtons [data-range]").forEach((btn) => {
    btn.addEventListener("click", () => {
      state.timelineRange = btn.dataset.range;
      updateTimelineControlStates();
      autoSelectTimelineView();
      renderDetailedTimeline();
    });
  });

  document.querySelectorAll("#viewModeButtons [data-view]").forEach((btn) => {
    btn.addEventListener("click", () => {
      if (btn.disabled) return;
      state.timelineView = btn.dataset.view;
      updateTimelineControlStates();
      renderDetailedTimeline();
    });
  });

  const customStart = document.getElementById("customStart");
  const customEnd = document.getElementById("customEnd");
  const onCustomChange = () => {
    if (state.timelineRange !== "custom") return;
    autoSelectTimelineView();
    renderDetailedTimeline();
  };
  customStart.addEventListener("change", onCustomChange);
  customEnd.addEventListener("change", onCustomChange);

  updateTimelineControlStates();
}

function getTimelineRangeDays() {
  if (state.timelineRange === "7d") return 7;
  if (state.timelineRange === "30d") return 30;
  if (state.timelineRange === "90d") return 90;
  if (state.timelineRange !== "custom") return 7;

  const start = document.getElementById("customStart").value;
  const end = document.getElementById("customEnd").value;
  if (!start || !end) return 7;

  const startMs = new Date(`${start}T00:00:00`).getTime();
  const endMs = new Date(`${end}T23:59:59`).getTime();
  if (Number.isNaN(startMs) || Number.isNaN(endMs) || endMs < startMs) return 7;

  const days = Math.ceil((endMs - startMs) / 86400000) + 1;
  return Math.max(1, days);
}

function getTimelineRangeBounds() {
  const now = Date.now();
  if (state.timelineRange === "7d") return { start: now - 7 * 86400000, end: now };
  if (state.timelineRange === "30d") return { start: now - 30 * 86400000, end: now };
  if (state.timelineRange === "90d") return { start: now - 90 * 86400000, end: now };

  const start = document.getElementById("customStart").value;
  const end = document.getElementById("customEnd").value;
  if (!start || !end) return { start: now - 7 * 86400000, end: now };

  const startMs = new Date(`${start}T00:00:00`).getTime();
  const endMs = new Date(`${end}T23:59:59`).getTime();
  if (Number.isNaN(startMs) || Number.isNaN(endMs) || endMs < startMs) return { start: now - 7 * 86400000, end: now };
  return { start: startMs, end: endMs };
}

function autoSelectTimelineView() {
  const rangeDays = getTimelineRangeDays();
  state.timelineView = rangeDays <= 7 ? "hourly" : "daily";
  updateTimelineControlStates();
}

function updateTimelineControlStates() {
  const customInputs = document.getElementById("customRangeInputs");
  customInputs.classList.toggle("hidden", state.timelineRange !== "custom");

  document.querySelectorAll("#rangeButtons [data-range]").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.range === state.timelineRange);
  });

  const rangeDays = getTimelineRangeDays();
  const forceDaily = rangeDays > 7;

  document.querySelectorAll("#viewModeButtons [data-view]").forEach((btn) => {
    const view = btn.dataset.view;
    const disabled = view === "hourly" && forceDaily;
    btn.disabled = disabled;
    btn.classList.toggle("active", state.timelineView === view);
  });
}

function setLoading(loading, message = "Loading dashboard data...") {
  const loadingEl = document.getElementById("globalLoading");
  const msgEl = document.getElementById("loadingMessage");
  if (!loadingEl || !msgEl) return;

  msgEl.textContent = message;
  if (loading) {
    loadingEl.classList.remove("hidden");
  } else {
    loadingEl.classList.add("hidden");
  }
}

function setPageStatus(message, type = "info") {
  const el = document.getElementById("pageStatus");
  if (!el) return;

  if (!message) {
    el.className = "page-status hidden";
    el.textContent = "";
    return;
  }

  el.textContent = message;
  el.className = `page-status ${type}`;
}

async function fetchJSONOrThrow(url) {
  const res = await fetch(url);
  if (!res.ok) {
    let details = `${res.status}`;
    try {
      details = `${res.status} ${await res.text()}`;
    } catch (_err) {
      // Ignore response body parse errors.
    }
    throw new Error(`Request failed: ${url} -> ${details}`);
  }
  return res.json();
}

async function loadDashboardData(repoNameWithOwner, isRefresh = false) {
  const repoName = repoNameWithOwner.split("/")[1];
  setLoading(true, isRefresh ? "Refreshing data..." : "Loading dashboard data...");

  try {
    const [history, commits, files, fileBreakdown] = await Promise.all([
      fetchJSONOrThrow(`http://localhost:8080/history?repo=${repoName}`),
      fetchJSONOrThrow(`http://localhost:8080/commits?repo=${repoName}&limit=100`),
      fetchJSONOrThrow(`http://localhost:8080/files?repo=${repoName}`),
      fetchJSONOrThrow(`http://localhost:8080/file-breakdown?repo=${repoName}`)
    ]);

    latestHistory = history || [];
    allCommits = Array.isArray(commits) ? commits : [];
    latestFiles = Array.isArray(files) ? files : [];
    latestBreakdown = fileBreakdown || {};

    allData = {
      repository: repo,
      timestamp: new Date().toISOString(),
      filters: {
        timeRangeDays: state.timeRangeDays,
        query: state.query
      },
      stats: {
        totalCommits: allCommits.length,
        totalFiles: latestFiles.length,
        activeFiles: latestFiles.filter((f) => f.status === "active").length
      },
      commits: allCommits,
      files: latestFiles,
      fileBreakdown: latestBreakdown
    };

    applyCommitFiltersAndRender();
    renderFileBreakdown(latestBreakdown);

    const now = new Date();
    document.getElementById("repoMeta").textContent = `Last refreshed ${formatRelativeTime(now.toISOString())}`;

    setPageStatus("", "info");
  } catch (err) {
    console.error("Failed to load dashboard data:", err);
    setPageStatus("Unable to load dashboard data. Check backend and retry.", "error");
  } finally {
    setLoading(false);
  }
}

function applyCommitFiltersAndRender() {
  filteredCommits = applyCommitFilters(allCommits, state.timeRangeDays, state.query);
  sortCommits(filteredCommits, state.sortBy, state.sortDir);

  updateVisibleSummary(filteredCommits.length, allCommits.length);
  updateStats(latestHistory, filteredCommits, allCommits, latestFiles);
  renderDetailedTimeline();
  renderCommitsPerDay(filteredCommits);
  renderContributionDistribution(filteredCommits);
  renderCommitTable(currentPage);
}

function applyCommitFilters(commits, timeRangeDays, query) {
  let result = Array.isArray(commits) ? [...commits] : [];

  if (timeRangeDays > 0) {
    const threshold = Date.now() - timeRangeDays * 24 * 60 * 60 * 1000;
    result = result.filter((c) => new Date(c.date).getTime() >= threshold);
  }

  if (query) {
    result = result.filter((c) => {
      const sha = (c.sha || "").toLowerCase();
      const author = (c.author || "").toLowerCase();
      const message = (c.message || "").toLowerCase();
      return sha.includes(query) || author.includes(query) || message.includes(query);
    });
  }

  return result;
}

function sortCommits(commits, sortBy, sortDir) {
  const direction = sortDir === "asc" ? 1 : -1;

  commits.sort((a, b) => {
    if (sortBy === "date") {
      return (new Date(a.date) - new Date(b.date)) * direction;
    }

    const aVal = (a[sortBy] || "").toString().toLowerCase();
    const bVal = (b[sortBy] || "").toString().toLowerCase();
    return aVal.localeCompare(bVal) * direction;
  });
}

function updateSortIndicators() {
  document.querySelectorAll(".sortable").forEach((th) => {
    th.classList.remove("sorted-asc", "sorted-desc");
    if (th.dataset.sort === state.sortBy) {
      th.classList.add(state.sortDir === "asc" ? "sorted-asc" : "sorted-desc");
    }
  });
}

function updateVisibleSummary(visible, total) {
  const el = document.getElementById("visibleSummary");
  if (!el) return;

  const rangeLabel = state.timeRangeDays > 0 ? `${state.timeRangeDays}d window` : "all time";
  const queryLabel = state.query ? `, query "${state.query}"` : "";
  el.textContent = `${visible}/${total} commits visible (${rangeLabel}${queryLabel})`;
}

function updateStats(history, visibleCommits, totalCommits, files) {
  const activityScoreEl = document.getElementById("activityScore");
  const totalCommitsEl = document.getElementById("totalCommits");
  const activeFilesEl = document.getElementById("activeFiles");
  const totalFilesEl = document.getElementById("totalFiles");

  if (history?.length > 0) {
    const avgScore = history.reduce((sum, h) => sum + h.score, 0) / history.length;
    activityScoreEl.textContent = Math.round(avgScore);
    document.getElementById("activityScoreSub").textContent = `Based on ${history.length} snapshots`;
  } else {
    activityScoreEl.textContent = "--";
    document.getElementById("activityScoreSub").textContent = "No snapshot history yet";
  }

  totalCommitsEl.textContent = visibleCommits?.length || 0;
  document.getElementById("totalCommitsSub").textContent =
    `${totalCommits?.length || 0} total in repository`;

  const activeCount = files ? files.filter((f) => f.status === "active").length : 0;
  activeFilesEl.textContent = activeCount;
  totalFilesEl.textContent = files?.length || 0;

  document.getElementById("activeFilesSub").textContent = "Modified in last 7 days";
  document.getElementById("totalFilesSub").textContent = "Tracked by sync history";
}

function toggleChartEmptyState(canvasId, emptyId, showEmpty) {
  const canvas = document.getElementById(canvasId);
  const empty = document.getElementById(emptyId);
  if (!canvas || !empty) return;

  canvas.style.opacity = showEmpty ? "0.12" : "1";
  empty.classList.toggle("hidden", !showEmpty);
}

function destroyChart(chartInstance) {
  if (chartInstance) {
    chartInstance.destroy();
  }
}

function renderDetailedTimeline() {
  const ctx = document.getElementById("timelineChart");
  if (!ctx) return;

  destroyChart(timelineChart);

  const { start, end } = getTimelineRangeBounds();
  const commits = applyCommitFilters(allCommits, 0, state.query)
    .filter((c) => {
      const ts = new Date(c.date).getTime();
      return ts >= start && ts <= end;
    })
    .sort((a, b) => new Date(a.date) - new Date(b.date));

  if (!commits.length) {
    toggleChartEmptyState("timelineChart", "timelineEmpty", true);
    return;
  }

  toggleChartEmptyState("timelineChart", "timelineEmpty", false);

  const bucketMs = state.timelineView === "hourly" ? 3600000 : 86400000;
  const bucketStart = Math.floor(start / bucketMs) * bucketMs;
  const bucketEnd = Math.floor(end / bucketMs) * bucketMs;
  const bucketCounts = new Map();

  for (let ts = bucketStart; ts <= bucketEnd; ts += bucketMs) {
    bucketCounts.set(ts, 0);
  }

  commits.forEach((c) => {
    const ts = new Date(c.date).getTime();
    const bucket = Math.floor(ts / bucketMs) * bucketMs;
    bucketCounts.set(bucket, (bucketCounts.get(bucket) || 0) + 1);
  });

  const points = [...bucketCounts.entries()].map(([x, y]) => ({ x, y }));
  const rangeDays = getTimelineRangeDays();
  const unit = state.timelineView === "hourly" ? "hour" : "day";

  timelineChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: state.timelineView === "hourly" ? "Commits per Hour" : "Commits per Day",
          data: points,
          backgroundColor: THEME.primarySoft,
          borderColor: THEME.primary,
          borderWidth: 2,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: THEME.primary,
          pointBorderColor: "#ffffff",
          pointBorderWidth: 1.5,
          tension: 0.25,
          fill: false
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { intersect: false, mode: "nearest" },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: THEME.tooltip,
          padding: 12,
          cornerRadius: 8,
          callbacks: {
            title: (items) => new Date(items[0].raw.x).toLocaleString(),
            label: (item) => `${state.timelineView === "hourly" ? "Commits/hour" : "Commits/day"}: ${item.parsed.y}`
          }
        },
        zoom: {
          zoom: {
            wheel: { enabled: false, speed: 0.1 },
            pinch: { enabled: false },
            mode: "x",
            onZoomComplete: () => {}
          },
          pan: {
            enabled: false,
            mode: "x",
            onPanComplete: () => {}
          },
          limits: { x: { min: "original", max: "original" } }
        }
      },
      scales: {
        x: {
          type: "time",
          min: bucketStart,
          max: bucketEnd,
          time: {
            unit,
            minUnit: "minute",
            tooltipFormat: "MMM d, yyyy HH:mm"
          },
          title: {
            display: true,
            text: `${state.timelineView === "hourly" ? "Hourly" : "Daily"} view (${rangeDays} days)`
          },
          grid: { color: THEME.grid },
          ticks: {
            color: THEME.textMuted,
            maxRotation: 45,
            minRotation: 0,
            autoSkip: true,
            maxTicksLimit: 12
          }
        },
        y: {
          beginAtZero: true,
          title: { display: true, text: "Commit Frequency (per hour)" },
          grid: { color: THEME.grid },
          ticks: { color: THEME.textMuted, stepSize: 1, precision: 0 }
        }
      }
    }
  });
}

function renderCommitsPerDay(commits) {
  const ctx = document.getElementById("commitsPerDayChart")?.getContext("2d");
  if (!ctx) return;

  destroyChart(commitsPerDayChart);

  if (!commits?.length) {
    toggleChartEmptyState("commitsPerDayChart", "commitsPerDayEmpty", true);
    return;
  }

  toggleChartEmptyState("commitsPerDayChart", "commitsPerDayEmpty", false);

  const countsByDay = new Map();
  commits.forEach((c) => {
    const key = new Date(c.date).toISOString().split("T")[0];
    countsByDay.set(key, (countsByDay.get(key) || 0) + 1);
  });

  const rows = [...countsByDay.entries()]
    .sort((a, b) => new Date(a[0]) - new Date(b[0]))
    .slice(-30);

  commitsPerDayChart = new Chart(ctx, {
    type: "bar",
    data: {
      labels: rows.map(([date]) => new Date(date).toLocaleDateString("en-US", { month: "short", day: "numeric" })),
      datasets: [
        {
          label: "Commits",
          data: rows.map(([, count]) => count),
          backgroundColor: "rgba(47, 154, 119, 0.55)",
          borderColor: THEME.success,
          borderWidth: 2
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        y: {
          beginAtZero: true,
          grid: { color: THEME.grid },
          ticks: { color: THEME.textMuted, stepSize: 1 }
        },
        x: {
          grid: { display: false },
          ticks: { color: THEME.textMuted }
        }
      }
    }
  });
}

function renderContributionDistribution(commits) {
  const ctx = document.getElementById("contributionChart")?.getContext("2d");
  if (!ctx) return;

  destroyChart(contributionChart);

  if (!commits?.length) {
    toggleChartEmptyState("contributionChart", "contributionEmpty", true);
    return;
  }

  toggleChartEmptyState("contributionChart", "contributionEmpty", false);

  const byAuthor = new Map();
  commits.forEach((c) => {
    const author = c.author || "Unknown";
    byAuthor.set(author, (byAuthor.get(author) || 0) + 1);
  });

  const contributors = [...byAuthor.entries()]
    .map(([author, count]) => ({ author, commits: count }))
    .sort((a, b) => b.commits - a.commits)
    .slice(0, 8);

  const colors = [
    "#1f7a8c", "#2f9a77", "#ef8354", "#4f6d80",
    "#4ca3b7", "#f2a65a", "#7fb685", "#35627c"
  ];

  contributionChart = new Chart(ctx, {
    type: "doughnut",
    data: {
      labels: contributors.map((c) => c.author),
      datasets: [
        {
          data: contributors.map((c) => c.commits),
          backgroundColor: colors,
          borderWidth: 2,
          borderColor: "#fff"
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          position: "right",
          labels: { color: THEME.textMuted }
        },
        tooltip: {
          callbacks: {
            label: (item) => {
              const total = item.dataset.data.reduce((a, b) => a + b, 0);
              const pct = ((item.parsed / total) * 100).toFixed(1);
              return `${item.label}: ${item.parsed} commits (${pct}%)`;
            }
          }
        }
      }
    }
  });
}

function renderCommitTable(page) {
  const tbody = document.getElementById("commitTableBody");
  const start = (page - 1) * commitsPerPage;
  const end = start + commitsPerPage;
  const pageCommits = filteredCommits.slice(start, end);

  if (!pageCommits.length) {
    tbody.innerHTML = '<tr><td colspan="4" class="empty-state">No commits match the current filters</td></tr>';
    document.getElementById("pageInfo").textContent = "Page 0 of 0";
    document.getElementById("prevPage").disabled = true;
    document.getElementById("nextPage").disabled = true;
    return;
  }

  tbody.innerHTML = pageCommits
    .map((c) => {
      const d = new Date(c.date);
      const dateStr = d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
      const timeStr = d.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" });
      const sha = (c.sha || "N/A").substring(0, 7);

      return `
      <tr>
        <td><span class="commit-sha">${sha}</span></td>
        <td>${dateStr} ${timeStr}<div class="table-subtext">${formatRelativeTime(c.date)}</div></td>
        <td>${highlightMatch(c.author || "Unknown", state.query)}</td>
        <td>${highlightMatch(c.message || "", state.query)}</td>
      </tr>`;
    })
    .join("");

  const totalPages = Math.ceil(filteredCommits.length / commitsPerPage);
  document.getElementById("pageInfo").textContent = `Page ${page} of ${totalPages}`;
  document.getElementById("prevPage").disabled = page <= 1;
  document.getElementById("nextPage").disabled = page >= totalPages;
}

function changePage(delta) {
  const totalPages = Math.max(1, Math.ceil(filteredCommits.length / commitsPerPage));
  currentPage += delta;
  if (currentPage < 1) currentPage = 1;
  if (currentPage > totalPages) currentPage = totalPages;
  renderCommitTable(currentPage);
}

function renderFileBreakdown(breakdown) {
  renderFileList("mostModifiedList", breakdown.most_modified || []);
  renderFileList("inactiveList", breakdown.inactive || []);
  renderFileList("frequentlyUpdatedList", breakdown.frequently_updated || []);
}

function renderFileList(id, files) {
  const el = document.getElementById(id);
  if (!files?.length) {
    el.innerHTML = '<div class="empty-state"><div class="icon">📂</div><p>No files found</p></div>';
    return;
  }

  el.innerHTML = files
    .map((f) => {
      const d = new Date(f.last_modified);
      const dateStr = d.toLocaleDateString("en-US", { month: "short", day: "numeric" });

      return `
      <div class="file-item ${f.status}">
        <div class="file-name">${escapeHTML(f.name)}</div>
        <div class="file-meta">
          <span class="badge ${f.status}">${f.status.toUpperCase()}</span>
          <span>${f.commit_count} commits</span>
          <span>Last: ${dateStr}</span>
        </div>
      </div>`;
    })
    .join("");
}

document.querySelectorAll(".tabs .tab").forEach((btn) => {
  btn.addEventListener("click", () => {
    const tabName = btn.dataset.tab;

    document.querySelectorAll(".tab").forEach((t) => t.classList.remove("active"));
    document.querySelectorAll(".tab-content").forEach((c) => c.classList.remove("active"));

    btn.classList.add("active");
    document.getElementById(tabName).classList.add("active");
  });
});

function exportData(format) {
  const repoName = repo.split("/")[1];
  const filename = `gitsense-${repoName}-${new Date().toISOString().split("T")[0]}`;

  if (format === "json") {
    const json = JSON.stringify(allData, null, 2);
    downloadFile(json, `${filename}.json`, "application/json");
  } else if (format === "csv") {
    const headers = ["SHA", "Date", "Author", "Message"];
    const rows = filteredCommits.map((c) => [
      c.sha || "",
      c.date,
      c.author,
      `"${(c.message || "").replace(/"/g, '""')}"`
    ]);

    const csv = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n");
    downloadFile(csv, `${filename}.csv`, "text/csv");
  }
}

function downloadFile(content, filename, mime) {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

function resetTimelineZoom() {
  state.timelineRange = "7d";
  autoSelectTimelineView();
  updateTimelineControlStates();
  renderDetailedTimeline();
  const info = document.getElementById("zoomInfo");
  info.textContent = "Reset to 7D hourly view";
  setTimeout(() => {
    info.textContent = "";
  }, 2500);
}

function highlightMatch(text, query) {
  const safe = escapeHTML(text || "");
  if (!query) return safe;

  const re = new RegExp(`(${escapeRegExp(query)})`, "ig");
  return safe.replace(re, "<mark>$1</mark>");
}

function escapeRegExp(text) {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function escapeHTML(text) {
  return String(text)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function formatRelativeTime(dateString) {
  const now = Date.now();
  const value = new Date(dateString).getTime();
  const diffMs = now - value;
  const diffMinutes = Math.floor(diffMs / 60000);

  if (diffMinutes < 1) return "just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 30) return `${diffDays}d ago`;

  const diffMonths = Math.floor(diffDays / 30);
  return `${diffMonths}mo ago`;
}
