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

// Delegated click-to-expand for truncated message rows. Attach once per
// container - survives innerHTML replacement since the listener lives on
// the container, not the rows themselves.
export function enableExpandableRows(container, messageSelector = ".event-message") {
    container.addEventListener("click", e => {
        const msg = e.target.closest(messageSelector);
        if (msg) msg.classList.toggle("expanded");
    });
}