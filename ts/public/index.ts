import { Action } from "svelte/action";
import { load, state } from "../shared/appstate.js";

type Preload = "mount" | "tap" | "hover";

/**
 * A svelte action to use inside <a> tags.
 * Will cause the link to preload and render on the client.
 */
export const preload = ((a: HTMLAnchorElement, preload: Preload = "hover") => {
    async function loadAnchor() {
        if (a.origin !== location.origin) return;
        if (a.href in state.hrefMap) return;
        state.hrefMap[a.href] = load(a.href);
    }

    if (preload === "mount") loadAnchor();
    if (preload === "hover") a.addEventListener("mouseover", loadAnchor);
    if (preload === "tap") {
        a.addEventListener("mousedown", loadAnchor);
        a.addEventListener("touchstart", loadAnchor);
    }

    a.addEventListener("click", async (e) => {
        if (a.origin !== location.origin) return;
        e.preventDefault();
        await goto(a.href);
    });
}) satisfies Action<HTMLAnchorElement, Preload | undefined>;

/**
 * Navigate to the url without reloading the page.
 * This function should be called for internal URLs only.
 */
export async function goto(url: string | URL) {
    const href = typeof url === "string" ? new URL(url, location.href).href : url.href;
    await state.update(href);
    history.pushState(href, "", href);
}
