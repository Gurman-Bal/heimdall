import { initializeSources } from "./views/sources.js";
import { initializeRules } from "./views/rules.js";
import { initializeReports } from "./views/reports.js";
import { initializeActivity } from "./views/activity.js";
import { initializeOps } from "./views/ops.js";

const STORAGE_KEY = "heimdall_active_view";

const viewInitializers = {
    sources: initializeSources,
    rules: initializeRules,
    reports: initializeReports,
    activity: initializeActivity,
    ops: initializeOps,
};

export function initializeRouter() {

    document
        .querySelectorAll(".tab")
        .forEach(tab => {
            tab.addEventListener("click", () => activateView(tab.dataset.view));
        });

    const savedView = localStorage.getItem(STORAGE_KEY);
    const validViews = Array.from(document.querySelectorAll(".tab")).map(t => t.dataset.view);

    if (savedView && validViews.includes(savedView)) {
        activateView(savedView);
    }
    // If no saved view, whatever's marked "active" in the HTML (watch) stays as-is.
}

function activateView(view) {

    document
        .querySelectorAll(".tab")
        .forEach(t => t.classList.toggle("active", t.dataset.view === view));

    document
        .querySelectorAll(".view")
        .forEach(v => v.classList.toggle("active", v.id === `view-${view}`));

    localStorage.setItem(STORAGE_KEY, view);

    const init = viewInitializers[view];
    if (init) init();
}