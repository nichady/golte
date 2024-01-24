<script>
    import { readonly, writable } from "svelte/store";
    import { Node } from "./node-wrapper.js";
    import { onMount, setContext } from "svelte";
    import { golteContext} from "./keys.js";
    import { fromArray } from "./list.js";
    import { get } from "svelte/store";
    import { AppState, load } from "./appstate.js";

    /** @typedef {import("./types.js").CompState} CompState */
    /** @typedef {import("./types.js").ContextData} ContextData */

    /** @type CompState[] */
    export let nodes;

    /** @type Promise<CompState[]> */
    export let promise;

    /** @type ContextData */
    export let contextData;

    const node = fromArray(nodes);

    const state = new AppState(contextData.URL, node, promise);

    setContext(golteContext, {
        url: readonly(state.url),
    });


    onMount(() => {
        history.replaceState(get(state.url).href, "");

        /** @type {(this: HTMLAnchorElement) => Promise<void>} */ 
        async function on() {
            if (this.href in state.hrefMap) return;
            state.hrefMap[this.href] = load(this.href);
        };

        for (const a of (document.querySelectorAll(`a[noreload="mount"]`))) {
            if (!(a instanceof HTMLAnchorElement)) continue;
            if (a.origin !== location.origin) continue;
            on.call(a)
        }

        for (const a of (document.querySelectorAll(`a[noreload="hover"]`))) {
            if (!(a instanceof HTMLAnchorElement)) continue;
            if (a.origin !== location.origin) continue;
            a.addEventListener("mouseover", on);
        }
        
        for (const a of (document.querySelectorAll(`a[noreload="tap"]`))) {
            if (!(a instanceof HTMLAnchorElement)) continue;
            if (a.origin !== location.origin) continue;
            a.addEventListener("mousedown", on);
            a.addEventListener("touchstart", on);
        }

        for (const a of (document.querySelectorAll(`a[noreload]`))) {
            if (!(a instanceof HTMLAnchorElement)) continue;
            if (a.origin !== location.origin) continue;
            a.addEventListener("click", async (e) => {
                e.preventDefault();
                await state.update(a.href);
                history.pushState(a.href, "", a.href);
            })
        }

        window.addEventListener("popstate", async (e) => {
            if (!e.state) return;
            await state.update(e.state);
        });
    });
</script>

<!-- #key is needed because csr error handling relies on constructor being called again -->
{#key $node}
    {#if $node}
        <Node node={$node} index={0} />
    {/if}
{/key}
