<script>
    import { readonly, writable } from "svelte/store";
    import { Node } from "./node-wrapper.js";
    import { onMount, setContext } from "svelte";
    import { golteContextKey, golteAnchorKey } from "./keys.js"
    import { fromArray } from "./list.js"
    import { get } from "svelte/store"

    export let nodes;
    export let contextData;

    const node = fromArray(nodes);

    const url = writable(new URL(contextData.URL));
    setContext(golteContextKey, {
        url: readonly(url),
    });

    async function onclick(e) {
        if (!(e.target instanceof HTMLAnchorElement)) return;
        if (e.target.origin !== location.origin) return;
        if (!e.target.hasAttribute("noreload")) return;

        e.preventDefault();
        const json = e.target[golteAnchorKey] ?? await load(e.target.href);
        await update(json, e.target.href);
        history.pushState({ golte: json, url: e.target.href }, "", e.target.href);
    }

    async function onhover(e) {
        if (!(e.target instanceof HTMLAnchorElement)) return;
        if (e.target.origin !== location.origin) return;
        if (e.target[golteAnchorKey]) return;
        if (e.target.getAttribute("noreload") !== "hover") return;

        const json = await load(e.target.href);
        e.target[golteAnchorKey] = json;
    }

    async function ontap(e) {
        if (!(e.target instanceof HTMLAnchorElement)) return;
        if (e.target.origin !== location.origin) return;
        if (e.target[golteAnchorKey]) return;
        if (e.target.getAttribute("noreload") !== "tap") return;

        const json = await load(e.target.href);
        e.target[golteAnchorKey] = json;
    }

    onMount(async () => {
        for (const a of document.querySelectorAll(`a[noreload="mount"]`)) {
            if (a.origin !== location.origin) return;
            const json = await load(a.href);
            a[golteAnchorKey] = json;
        }
    });

    async function onpopstate(e) {
        if (!e.state || !e.state.golte) return
        await update(e.state.golte, e.state.url);
    };

    async function load(href) {
        const headers = {
            "Golte": "true",
        };

        const resp = await fetch(href, { headers });
        const json = await resp.json();

        for (const entry of [...json.Entries, json.ErrPage]) {
            // preload js
            import(entry.File);

            // preload css
            for (const css of entry.CSS) {
                if (document.querySelector(`link[href="${css}"][rel="stylesheet"]`)) continue;
                const link = document.createElement("link");
                link.href = css;
                link.rel = "stylesheet";
                document.head.appendChild(link);
            }
        }
        
        return json;
    }

    /**
     * @param {any[]} json
     */
    async function update(json, u) {
        const promises = json.Entries.map(async (entry) => ({
            comp: (await import(entry.File)).default,
            props: entry.Props,
            errPage: (await import(json.ErrPage.File)).default,
        }));
        
        $url = new URL(u);

        // this loop replaces the first differentiated node from after onto before
        // the reason this is done instead of simply replacing the first node is so we don't rerender unnecessary nodes
        // this allows for data persistence in already rendered nodes
        let before = node;
        let after = fromArray(await Promise.all(promises));
        while (true) {
            const bval = get(before);
            const aval = get(after);

            if (!bval && !aval) break; // both nodes are null - end of list

            const bcomp = bval.content.comp;
            const acomp = aval.content.comp;

            if (bcomp === acomp) { // nodes are same component - pass
                before = bval.next;
                after = aval.next;
            } else { // nodes are different components - replace
                before.set(aval);
                break;
            }
        }
    }
</script>

<svelte:document on:click={onclick} on:mouseover={onhover} on:mousedown={ontap} on:touchstart={ontap} />
<svelte:window on:popstate={onpopstate} />

<!-- #key is needed because csr error handling relies on constructor being called again -->
{#key $node}
    {#if $node}
        <Node node={$node} index={0} />
    {/if}
{/key}
