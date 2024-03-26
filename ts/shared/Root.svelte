<script>
    import { Node } from "./node-wrapper.js";
    import { onMount } from "svelte";
    import { get } from "svelte/store";
    import { initState, state } from "./appstate.js";

    /** @type {import("./types.js").CompState[]} */
    export let nodes;

    /** @type {import("./types.js").ContextData} */
    export let contextData;

    initState(contextData.URL, nodes);
    const { node } = state;

    onMount(() => {
        history.replaceState(get(state.url).href, "");
        addEventListener("popstate", async (e) => {
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
