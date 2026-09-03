import { login } from "./api.js";

const overlay = document.getElementById("login-overlay");
const form = document.getElementById("login-form");
const error = document.getElementById("login-error");

export function showLogin() {
    overlay.classList.add("active");

    // Don't let the user interact with the dashboard
    // while authentication is required.
    document.body.classList.add("login-required");
}

export function hideLogin() {
    overlay.classList.remove("active");
    document.body.classList.remove("login-required");
}

export function initializeLogin() {
    if (!overlay || !form) {
        console.error("Login UI elements not found");
        return;
    }

    form.addEventListener("submit", async (e) => {
        e.preventDefault();

        error.textContent = "";

        const username =
            document.getElementById("login-username").value.trim();

        const password =
            document.getElementById("login-password").value;

        try {
            const res = await login(username, password);

            if (!res.ok) {
                error.textContent =
                    res.status === 401
                        ? "Invalid credentials"
                        : "Login failed";

                return;
            }

            hideLogin();

            // Reload so every view initializes with the new session.
            location.reload();

        } catch (err) {
            console.error("Login failed:", err);
            error.textContent = "Unable to connect to Heimdall";
        }
    });
}


// Install the global 401 handler.
//
// Any protected API request that receives 401 means:
// "your Heimdall session is gone."
const originalFetch = window.fetch;

window.fetch = async (...args) => {
    const res = await originalFetch(...args);

    const url = args[0]?.toString() ?? "";

    if (
        res.status === 401 &&
        !url.includes("/api/auth/login") &&
        !url.includes("/api/auth/logout")
    ) {
        showLogin();
    }

    return res;
};