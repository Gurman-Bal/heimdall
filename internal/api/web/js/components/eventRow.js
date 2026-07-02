import {
    escapeHtml,
    fmtTime
} from "../utils.js";

export function eventRow(e) {

    return `
        <div class="event-row">

            <span class="event-time">
                ${fmtTime(e.Timestamp)}
            </span>

            <span class="badge ${e.Severity}">
                ${e.Severity}
            </span>

            <span class="event-source">
                ${e.Source}
            </span>

            <span class="event-type">
                ${e.Type}
            </span>

            <span
                class="event-message"
                title="${escapeHtml(e.Message)}">

                ${escapeHtml(e.Message)}

            </span>

        </div>
    `;
}