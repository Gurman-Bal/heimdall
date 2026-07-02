import {
    getRules,
    getSourceTypes,
    addRule,
    removeRule
}
    from "../api.js";

import {
    ruleRow
}
    from "../components/ruleRow.js";

const ruleList =
    document.getElementById("rule-list");

const ruleForm =
    document.getElementById("rule-form");

const ruleError =
    document.getElementById("rule-error");

const ruleSourceType =
    document.getElementById(
        "rule-source-type"
    );

const rulePattern =
    document.getElementById(
        "rule-pattern"
    );

const ruleSeverity =
    document.getElementById(
        "rule-severity"
    );

const ruleEventType =
    document.getElementById(
        "rule-event-type"
    );

let initialized = false;

export async function initializeRules() {

    await loadSourceTypes();

    await loadRules();

    if (!initialized) {

        initializeForm();

        initialized = true;

    }
}

async function loadSourceTypes() {

    const types =
        await getSourceTypes();

    ruleSourceType.innerHTML =
        types
            .map(
                t =>
                    `<option value="${t}">
                        ${t}
                    </option>`
            )
            .join("");
}

async function loadRules() {

    const rules =
        await getRules();

    if (rules.length === 0) {

        ruleList.innerHTML = `
            <div class="empty-state">
                no rules configured
            </div>
        `;

        return;
    }

    ruleList.innerHTML =
        rules
            .map(ruleRow)
            .join("");

    ruleList
        .querySelectorAll(".remove-btn")
        .forEach(btn => {

            btn.addEventListener(
                "click",
                async () => {

                    await removeRule(
                        btn.dataset.id
                    );

                    loadRules();

                }
            );

        });

}

function initializeForm() {

    ruleForm.addEventListener(
        "submit",
        async e => {

            e.preventDefault();

            ruleError.textContent = "";

            const body = {

                type:
                ruleSourceType.value,

                pattern:
                rulePattern.value,

                severity:
                ruleSeverity.value,

                event_type:
                ruleEventType.value,

                priority: 100

            };

            const res =
                await addRule(body);

            if (!res.ok) {

                ruleError.textContent =
                    await res.text();

                return;
            }

            rulePattern.value = "";

            ruleEventType.value = "";

            loadRules();

        }
    );

}