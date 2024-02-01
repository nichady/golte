import { getContext } from "svelte";
import { Action } from "svelte/action";
import { golteContext } from "../shared/keys.js";
import { AppState, load } from "../shared/appstate.js";

type Preload = "mount" | "tap" | "hover";

export const preload = (a: HTMLAnchorElement, preload: Preload = "hover") => {
    const state: AppState = getContext(golteContext);

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
        await state.update(a.href);
        history.pushState(a.href, "", a.href);
    });
};

// this only serves as a typecheck for now until typescript supports "satisfies" for functions directly
let _ = preload satisfies Action<HTMLAnchorElement, Preload | undefined>;
