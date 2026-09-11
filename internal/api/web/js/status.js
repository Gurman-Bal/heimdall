import { getSystemStatus } from "./api.js";

const appStateDot = document.getElementById("app-state-dot");
const appStateText = document.getElementById("app-state-text");

export function initializeStatusPolling() {
    poll();
    setInterval(poll, 5000);
}

async function poll() {
    const status = await getSystemStatus();
    if (!status) return;

    const worker = status.worker || {};
    const state = worker.state || "unreachable";

    appStateDot.className = `status-dot ${stateClass(state)}`;
    appStateText.textContent = state;
}

function stateClass(state) {
    if (state === "unreachable") return "critical";
    if (state === "restarting" || state === "stopping") return "warning";
    return "info";
}