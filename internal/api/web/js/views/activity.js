import { getActivity } from "../api.js";

const activityList = document.getElementById("activity-list");
const pauseBtn = document.getElementById("activity-pause-btn");

let currentWindow = "1h";
let initialized = false;
let activityPoller = null;
let expandedKey = null;
let paused = false;

export async function initializeActivity() {

    await loadActivity();

    if (!initialized) {
        initializeFilters();
        activityList.addEventListener("click", handleRowClick);
        pauseBtn.addEventListener("click", togglePause);
        initialized = true;
    }

    if (activityPoller) clearInterval(activityPoller);
    activityPoller = setInterval(() => {
        if (!paused) loadActivity();
    }, 5000);
}

export function teardownActivity() {
    if (activityPoller) {
        clearInterval(activityPoller);
        activityPoller = null;
    }
}

function togglePause() {
    paused = !paused;
    pauseBtn.textContent = paused ? "RESUME" : "PAUSE";
    pauseBtn.classList.toggle("active", paused);
}

function handleRowClick(e) {
    const row = e.target.closest(".activity-row");
    if (!row) return;
    const key = row.dataset.key;
    expandedKey = expandedKey === key ? null : key;
    applyExpandedState();
}

function applyExpandedState() {
    activityList.querySelectorAll(".activity-row").forEach(row => {
        row.querySelector(".event-message")
            ?.classList.toggle("expanded", row.dataset.key === expandedKey);
    });
}

function initializeFilters() {
    document.querySelectorAll("#view-activity .filter-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            document.querySelectorAll("#view-activity .filter-btn")
                .forEach(b => b.classList.remove("active"));
            btn.classList.add("active");
            currentWindow = btn.dataset.window;
            loadActivity();
        });
    });
}

async function loadActivity() {

    const entries = await getActivity(currentWindow);

    if (!entries || entries.length === 0) {
        activityList.innerHTML = `<div class="empty-state">no activity recorded in this window</div>`;
        return;
    }

    activityList.innerHTML = entries
        .map(e => {
            const key = `${e.Time}-${e.Message}`;
            return `
                <div class="activity-row ${(e.Level || "").toLowerCase()}" data-key="${key}">
                    <span class="event-time">${new Date(e.Time).toLocaleString()}</span>
                    <span class="activity-level">${e.Level}</span>
                    <span class="event-message">${e.Message}</span>
                </div>
            `;
        })
        .join("");

    applyExpandedState();
}