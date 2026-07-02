import { escapeHtml } from "../utils.js";

export function ruleRow(r) {

    return `
        <div class="source-row">

            <div class="source-info">

                <span class="source-type">
                    ${r.SourceType}
                </span>

                <span class="badge ${r.Severity}">
                    ${r.Severity}
                </span>

                <span class="event-type">
                    ${r.EventType}
                </span>

                <span class="source-path">
                    ${escapeHtml(r.Pattern)}
                </span>

            </div>

            <button
                class="remove-btn"
                data-id="${r.ID}">

                REMOVE

            </button>

        </div>
    `;
}