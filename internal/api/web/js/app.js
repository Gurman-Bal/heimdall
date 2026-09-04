import { initializeRouter } from "./router.js";
import { initializeWatch } from "./views/watch.js";
import { initializeLogin, hideLogin, showLogin } from "./login.js";
import { getSystemStatus, logout } from "./api.js";
import { initializeStatusPolling } from "./status.js";

const logoutButton = document.getElementById("logout-btn");

logoutButton.addEventListener("click", async () => {
    try {
        await logout();
    } finally {
        location.reload();
    }
});

async function initializeApp() {
    // Install login UI + global 401 handling first.
    initializeLogin();

    // Check whether we already have a valid session.
    //
    // getSystemStatus() returns the parsed status object on success, or
    // null on failure — not a Response, so check truthiness, not `.ok`.
    try {
        const status = await getSystemStatus();

        if (status) {
            hideLogin();
        }
    } catch (err) {
        console.error("Failed to check authentication:", err);
    }

    initializeRouter();
    initializeWatch();
    initializeStatusPolling();
}

initializeApp();