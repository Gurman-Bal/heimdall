import { initializeSources } from "./views/sources.js";
import { initializeRules } from "./views/rules.js";
import { initializeReports } from "./views/reports.js";
import { initializeActivity, teardownActivity } from "./views/activity.js";
import { initializeOps, teardownOps } from "./views/ops.js";

const STORAGE_KEY = "heimdall_active_view";

const viewInitializers = {
    sources: initializeSources,
    rules: initializeRules,
    reports: initializeReports,
    activity: initializeActivity,
    ops: initializeOps,
};

const viewTeardowns = {
    activity: teardownActivity,
    ops: teardownOps,
};

let currentView = "watch";

export function initializeRouter() {

    document.querySelectorAll(".tab").forEach(tab => {
        tab.addEventListener("click", () => activateView(tab.dataset.view));
    });

    const savedView = localStorage.getItem(STORAGE_KEY);
    const validViews = Array.from(document.querySelectorAll(".tab")).map(t => t.dataset.view);

    if (savedView && validViews.includes(savedView)) {
        activateView(savedView);
    }
}

function activateView(view) {

    const teardown = viewTeardowns[currentView];
    if (teardown) teardown();

    document.querySelectorAll(".tab")
        .forEach(t => t.classList.toggle("active", t.dataset.view === view));

    document.querySelectorAll(".view")
        .forEach(v => v.classList.toggle("active", v.id === `view-${view}`));

    localStorage.setItem(STORAGE_KEY, view);
    currentView = view;

    const init = viewInitializers[view];
    if (init) init();
}