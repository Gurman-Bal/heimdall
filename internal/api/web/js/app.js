import { initializeRouter } from "./router.js";
import { initializeWatch } from "./views/watch.js";

import { logout } from "./api.js";
document.getElementById("logout-btn").addEventListener("click", async () => {
    await logout();
    location.reload();
});

initializeRouter();
initializeWatch();