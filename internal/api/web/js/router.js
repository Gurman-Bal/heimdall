import { initializeSources } from "./views/sources.js";
import { initializeRules } from "./views/rules.js";

export function initializeRouter() {

    document
        .querySelectorAll(".tab")
        .forEach(tab => {

            tab.addEventListener("click", async () => {

                document
                    .querySelectorAll(".tab")
                    .forEach(t => t.classList.remove("active"));

                document
                    .querySelectorAll(".view")
                    .forEach(v => v.classList.remove("active"));

                tab.classList.add("active");

                const view = tab.dataset.view;

                document
                    .getElementById(`view-${view}`)
                    .classList
                    .add("active");

                if (view === "sources") {
                    await initializeSources();
                }

                if (view === "rules") {
                    await initializeRules();
                }

            });

        });

}