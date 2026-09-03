import { getActivity } from "../api.js";

const activityList =
    document.getElementById("activity-list");

let poller = null;

export async function initializeActivity() {

    await loadActivity();

    if (!poller) {
        poller = setInterval(loadActivity, 5000);
    }
}

async function loadActivity() {

    const entries =
        await getActivity();

    if (entries.length === 0) {

        activityList.innerHTML = `
            <div class="empty-state">
                no activity recorded yet
            </div>
        `;

        return;
    }

    activityList.innerHTML =
        entries
            .slice()
            .reverse()
            .map(e => `
                <div class="activity-row ${e.level.toLowerCase()}">
                    <span class="event-time">
                        ${new Date(e.time).toLocaleTimeString()}
                    </span>
                    <span class="activity-level">
                        ${e.level}
                    </span>
                    <span class="event-message">
                        ${e.message}
                    </span>
                </div>
            `)
            .join("");
}