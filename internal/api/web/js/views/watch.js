import { getEvents } from "../api.js";
import { eventRow } from "../components/eventRow.js";

let events = [];
let currentFilter = "all";

const eventList = document.getElementById("event-list");

const statusDot = document.getElementById("status-dot");
const statusText = document.getElementById("status-text");
const bifrost = document.getElementById("bifrost");

export async function initializeWatch() {

    events = await getEvents();

    initializeFilters();

    renderEvents();

    updateStatus();

    connectStream();
}

function initializeFilters() {

    document
        .querySelectorAll(".filter-btn")
        .forEach(btn => {

            btn.addEventListener("click", () => {

                document
                    .querySelectorAll(".filter-btn")
                    .forEach(b => b.classList.remove("active"));

                btn.classList.add("active");

                currentFilter =
                    btn.dataset.severity;

                renderEvents();

            });

        });

}

function renderEvents() {

    const filtered =
        currentFilter === "all"
            ? events
            : events.filter(
                e => e.Severity === currentFilter
            );

    if (filtered.length === 0) {

        eventList.innerHTML = `
            <div class="empty-state">
                no events
                ${
            currentFilter !== "all"
                ? ` at ${currentFilter} severity`
                : " yet — heimdall is watching"
        }
            </div>
        `;

        return;
    }

    eventList.innerHTML =
        filtered
            .map(eventRow)
            .join("");
}

function updateStatus() {

    const hasCritical =
        events.some(
            e => e.Severity === "critical"
        );

    const hasWarning =
        events.some(
            e => e.Severity === "warning"
        );

    const level =
        hasCritical
            ? "critical"
            : hasWarning
                ? "warning"
                : "info";

    const label =
        hasCritical
            ? "critical events active"
            : hasWarning
                ? "warnings present"
                : "nominal";

    statusDot.className =
        `status-dot ${level}`;

    statusText.textContent =
        label;

    bifrost.className =
        `bifrost ${level}`;
}

function prependEvent(e) {

    events.unshift(e);

    if (events.length > 200) {
        events.pop();
    }

    renderEvents();

    updateStatus();
}

function connectStream() {

    const es =
        new EventSource("/api/stream");

    es.onmessage = msg => {

        const event =
            JSON.parse(msg.data);

        prependEvent(event);
    };

    es.onerror = () => {

        es.close();

        setTimeout(
            connectStream,
            3000
        );

    };
}