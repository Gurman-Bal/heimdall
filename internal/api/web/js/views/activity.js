import { getActivity } from "../api.js";

const activityList =
    document.getElementById("activity-list");

let currentWindow = "1h";
let initialized = false;

export async function initializeActivity() {

    await loadActivity();

    if (!initialized) {
        initializeFilters();
        initialized = true;
    }
}

function initializeFilters() {

    document
        .querySelectorAll("#view-activity .filter-btn")
        .forEach(btn => {

            btn.addEventListener("click", () => {

                document
                    .querySelectorAll("#view-activity .filter-btn")
                    .forEach(b => b.classList.remove("active"));

                btn.classList.add("active");

                currentWindow =
                    btn.dataset.window;

                loadActivity();

            });

        });

}

async function loadActivity() {

    const entries =
        await getActivity(currentWindow);

    if (!entries || entries.length === 0) {

        activityList.innerHTML = `
            <div class="empty-state">
                no activity recorded in this window
            </div>
        `;

        return;
    }

    activityList.innerHTML =
        entries
            .map(e => `
                <div class="activity-row ${(e.Level || "").toLowerCase()}">
                    <span class="event-time">
                        ${new Date(e.Time).toLocaleString()}
                    </span>
                    <span class="activity-level">
                        ${e.Level}
                    </span>
                    <span class="event-message">
                        ${e.Message}
                    </span>
                </div>
            `)
            .join("");
}