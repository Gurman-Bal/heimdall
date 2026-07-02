export async function getEvents() {
    return (await fetch("/api/events")).json();
}

export async function getSourceTypes() {
    return (await fetch("/api/source-types")).json();
}

export async function getSources() {
    return (await fetch("/api/sources")).json();
}

export async function addSource(body) {

    return fetch("/api/sources", {
        method: "POST",

        headers: {
            "Content-Type": "application/json"
        },

        body: JSON.stringify(body)
    });
}

export async function removeSource(id) {

    return fetch(`/api/sources/${id}`, {
        method: "DELETE"
    });
}

export async function getRules() {
    return (await fetch("/api/rules")).json();
}

export async function addRule(body) {

    return fetch("/api/rules", {
        method: "POST",

        headers: {
            "Content-Type": "application/json"
        },

        body: JSON.stringify(body)
    });
}

export async function removeRule(id) {

    return fetch(`/api/rules/${id}`, {
        method: "DELETE"
    });
}