import { escapeHtml } from "../utils.js";

function issueRow(issue) {

    const severityClass =
        issue.severity === "critical"
            ? "critical"
            : "warning";

    return `
        <div class="issue-row ${severityClass}">

            <div class="issue-title">
                ${escapeHtml(issue.title)}
            </div>

            <div class="issue-explanation">
                ${escapeHtml(issue.explanation)}
            </div>

            <div class="issue-fix">
                Suggested:
                ${escapeHtml(issue.suggested_fix)}
            </div>

        </div>
    `;
}

export function reportRow(r) {

    const issues =
        JSON.parse(r.IssuesJSON || "[]");

    const generatedAt =
        new Date(r.GeneratedAt)
            .toLocaleString();

    return `
        <div
            class="report-row"
            data-id="${r.ID}">

            <div class="report-header">

                <span class="event-time">
                    ${generatedAt}
                </span>

                <span class="source-type">
                    ${r.Model}
                </span>

                <span class="event-source">
                    ${r.EventCount} events
                </span>

            </div>

            <div class="report-summary">
                ${escapeHtml(r.Summary)}
            </div>

            ${issues.length > 0
        ? issues.map(issueRow).join("")
        : `<div class="empty-state">no issues detected this period</div>`
    }

        </div>
    `;
}