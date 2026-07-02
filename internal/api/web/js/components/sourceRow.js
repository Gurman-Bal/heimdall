import { escapeHtml } from "../utils.js";

export function sourceRow(s) {

    return `
        <div
            class="source-row"
            data-id="${s.ID}">

            <div class="source-info">

                <span class="source-type">
                    ${s.Type}
                </span>

                <span class="source-path">
                    ${escapeHtml(s.Path)}
                </span>

            </div>

            <button
                class="remove-btn"
                data-id="${s.ID}">

                REMOVE

            </button>

        </div>
    `;
}