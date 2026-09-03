import { runCommand, getContainers, changePassword } from "../api.js";

const containerButtons = document.getElementById("container-buttons");
const commandInput = document.getElementById("command-input");
const commandForm = document.getElementById("command-form");
const commandError = document.getElementById("command-error");
const passwordForm = document.getElementById("password-form");
const passwordError = document.getElementById("password-error");

let initialized = false;

export async function initializeOps() {

    await loadContainers();

    if (!initialized) {
        initializeForms();
        initialized = true;
    }
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
    }
}