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

    appStateDot.className = `status-dot ${stateClass(status.state)}`;
    appStateText.textContent = status.state;
}

function stateClass(state) {
    if (state === "restarting" || state === "stopping") return "warning";
    return "info";
}