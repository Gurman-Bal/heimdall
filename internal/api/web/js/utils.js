export function escapeHtml(str) {

    const div = document.createElement("div");

    div.textContent = str;

    return div.innerHTML;
}

export function fmtTime(ts) {

    return new Date(ts)
        .toLocaleTimeString(
            "en-GB",
            {
                hour12: false
            }
        );
}