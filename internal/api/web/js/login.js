import { login } from "./api.js";

const overlay = document.getElementById("login-overlay");
const form = document.getElementById("login-form");
const error = document.getElementById("login-error");

export function showLogin() {
    overlay.classList.add("active");
}

export function hideLogin() {
    overlay.classList.remove("active");
}

export function initializeLogin() {
    form.addEventListener("submit", async e => {
        e.preventDefault();
        error.textContent = "";

        const username = document.getElementById("login-username").value;
        const password = document.getElementById("login-password").value;

        const res = await login(username, password);
        if (!res.ok) {
            error.textContent = "Invalid credentials";
            return;
        }

        hideLogin();
        location.reload();
    });
}

// Wrap fetch globally so any 401 anywhere in the app triggers the login
// overlay — this is what makes session timeout actually visible to the user
// instead of API calls just silently failing.
const originalFetch = window.fetch;
window.fetch = async (...args) => {
    const res = await originalFetch(...args);
    if (res.status === 401 && !args[0].toString().includes("/api/auth/login")) {
        showLogin();
    }
    return res;
};