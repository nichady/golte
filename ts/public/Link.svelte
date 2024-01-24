<script>
    import { getContext, onMount } from "svelte";
    import { golteContext } from "../shared/keys";
    import { load as appstateLoad } from "../shared/appstate";

    /** @type {string} */
    export let href;

    /** @type {"mount" | "hover" | "tap"} */
    export let preload = "hover";

    /** @type {HTMLAnchorElement} */
    let a;

    /** @type {import("../shared/appstate").AppState} */
    const state = getContext(golteContext);

    async function load() {
        if (a.origin !== location.origin) return;
        if (a.href in state.hrefMap) return;
        state.hrefMap[a.href] = appstateLoad(a.href);
    }

    onMount(() => {
        if (preload === "mount") load();
    });

    function hover() {
        if (preload === "hover") load();
    }

    function tap() {
        if (preload === "tap") load();
    }

    /** @param {MouseEvent} e */
    async function click(e) {
        if (a.origin !== location.origin) return;
        e.preventDefault();
        await state.update(a.href);
        history.pushState(a.href, "", a.href);
    }
</script>

<!-- svelte-ignore a11y-mouse-events-have-key-events -->
<a
    {href}
    {...$$restProps}
    bind:this={a}
    on:mouseover={hover}
    on:mousedown={tap}
    on:touchstart={tap}
    on:click={click}
>
    <slot />
</a>
