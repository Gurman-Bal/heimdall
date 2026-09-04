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
    // If the user is already logged in, this succeeds and the
    // dashboard remains visible.
    //
    // If not, the global fetch handler sees 401 and opens login.
    try {
        const res = await getSystemStatus();

        if (res.ok) {
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