import { getContainers, changePassword, getSystemStatus, getSettings, updateSettings, containerAction } from "../api.js";

const systemStatusPanel = document.getElementById("system-status");
const containerButtons = document.getElementById("container-buttons");
const passwordForm = document.getElementById("password-form");
const passwordError = document.getElementById("password-error");
const settingsForm = document.getElementById("settings-form");
const settingsError = document.getElementById("settings-error");
const sessionTimeoutInput = document.getElementById("session-timeout-minutes");
const reconnectOverlay = document.getElementById("reconnect-overlay");

let initialized = false;

export async function initializeOps() {

    await loadSystemStatus();
    await loadContainers();
    await loadSettings();

    if (!initialized) {
        initializeForms();
        initialized = true;
    }
}

async function loadSettings() {
    const settings = await getSettings();
    if (settings) {
        sessionTimeoutInput.value = Math.round(settings.session_timeout_seconds / 60);
    }
}

async function loadSystemStatus() {

    const status = await getSystemStatus();

    if (!status) {
        systemStatusPanel.innerHTML = `<div class="empty-state">could not load system status</div>`;
        return;
    }

    systemStatusPanel.innerHTML = `
        <div class="source-row">
            <span class="source-path">state</span>
            <span class="source-type">${status.state}</span>
        </div>
        <div class="source-row">
            <span class="source-path">uptime</span>
            <span class="source-type">${formatUptime(status.uptime_seconds)}</span>
        </div>
        <div class="source-row">
            <span class="source-path">registered sources</span>
            <span class="source-type">${status.registered_types.join(", ") || "none"}</span>
        </div>
        <div class="source-row">
            <span class="source-path">events dropped (buffer full)</span>
            <span class="source-type">${status.events_dropped}</span>
        </div>
    `;
}

function formatUptime(seconds) {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return `${h}h ${m}m`;
}

async function loadContainers() {

    const containers = await getContainers();

    // Self can only be restarted, never stopped — stopping it from its own
    // UI would leave no way to start it back up from here.
    containerButtons.innerHTML =
        containers
            .map(c => `
                <div class="source-row">
                    <span class="source-path">${c.name}${c.is_self ? " (this instance)" : ""}</span>
                    <div>
                        <button class="remove-btn" data-name="${c.name}" data-action="restart" data-self="${c.is_self}">RESTART</button>
                        ${c.is_self ? "" : `
                            <button class="remove-btn" data-name="${c.name}" data-action="stop" data-self="false">STOP</button>
                            <button class="remove-btn" data-name="${c.name}" data-action="start" data-self="false">START</button>
                        `}
                    </div>
                </div>
            `)
            .join("");

    containerButtons.querySelectorAll("button").forEach(btn => {
        btn.addEventListener("click", () =>
            sendAction(btn.dataset.name, btn.dataset.action, btn.dataset.self === "true")
        );
    });
}

function initializeForms() {

    passwordForm.addEventListener("submit", async e => {
        e.preventDefault();
        passwordError.textContent = "";

        const current = document.getElementById("current-password").value;
        const next = document.getElementById("new-password").value;

        const res = await changePassword(current, next);
        if (!res.ok) {
            passwordError.textContent = await res.text();
            return;
        }

        passwordForm.reset();
        alert("Password changed.");
    });

    settingsForm.addEventListener("submit", async e => {
        e.preventDefault();
        settingsError.textContent = "";

        const minutes = parseInt(sessionTimeoutInput.value, 10);
        const res = await updateSettings(minutes * 60);

        if (!res.ok) {
            settingsError.textContent = await res.text();
            return;
        }

        alert("Settings saved.");
    });
}

async function sendAction(name, action, isSelf) {

    if (!confirm(`${action.toUpperCase()} "${name}"?`)) return;

    const res = await containerAction(name, action);

    if (!res.ok) {
        alert(await res.text());
        return;
    }

    if (isSelf && action === "restart") {
        // Heimdall's own process is about to exit and come back via Docker's
        // restart policy. The page can't talk to a dead process, so show a
        // waiting state and reload once it's reachable again, instead of
        // just erroring out or looking frozen.
        waitForReconnect();
    } else {
        setTimeout(loadSystemStatus, 500);
    }
}

function waitForReconnect() {

    reconnectOverlay.classList.add("active");

    const poll = setInterval(async () => {
        const status = await getSystemStatus();
        if (status) {
            clearInterval(poll);
            location.reload();
        }
    }, 1000);
}