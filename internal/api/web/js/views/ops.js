import { runCommand, getContainers, changePassword, getSystemStatus } from "../api.js";

const systemStatusPanel = document.getElementById("system-status");
const containerButtons = document.getElementById("container-buttons");
const commandInput = document.getElementById("command-input");
const commandForm = document.getElementById("command-form");
const commandError = document.getElementById("command-error");
const passwordForm = document.getElementById("password-form");
const passwordError = document.getElementById("password-error");

let initialized = false;

export async function initializeOps() {

    await loadSystemStatus();
    await loadContainers();

    if (!initialized) {
        initializeForms();
        initialized = true;
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

    const containers =
        await getContainers();

    containerButtons.innerHTML =
        containers
            .map(name => `
                <div class="source-row">
                    <span class="source-path">${name}</span>
                    <div>
                        <button class="remove-btn" data-cmd="restart ${name}">RESTART</button>
                        <button class="remove-btn" data-cmd="stop ${name}">STOP</button>
                        <button class="remove-btn" data-cmd="start ${name}">START</button>
                    </div>
                </div>
            `)
            .join("");

    containerButtons.querySelectorAll("button").forEach(btn => {
        btn.addEventListener("click", () => sendCommand(btn.dataset.cmd));
    });
}

function initializeForms() {

    commandForm.addEventListener("submit", async e => {
        e.preventDefault();
        await sendCommand(commandInput.value.trim());
        commandInput.value = "";
    });

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
        alert("Password changed. You may need to re-authenticate.");
    });
}

async function sendCommand(cmd) {

    commandError.textContent = "";

    if (!confirm(`Run "${cmd}"?`)) return;

    const res = await runCommand(cmd);

    if (!res.ok) {
        commandError.textContent = await res.text();
    } else {
        setTimeout(loadSystemStatus, 500);
    }
}