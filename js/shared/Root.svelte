<script>
    import { readonly, writable } from "svelte/store";
    import { Node } from "./node-wrapper.js";
    import { setContext } from "svelte";
    import { golteContextKey, golteAnchorKey } from "./keys.js"

    export let nodes;
    export let contextData;

    const url = writable(new URL(contextData.uRL));
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

        for (const entry of json) {
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
        const promises = json.map(async (entry) => ({
            comp: (await import(entry.File)).default,
            props: entry.Props,
        }));

        $url = new URL(u);
        nodes = await Promise.all(promises);
    }
</script>

{#if nodes[0]}
    <Node nodes={nodes} index={0} />
{/if}

<svelte:document on:click={onclick} on:mouseover={onhover} on:mousedown={ontap} on:touchstart={ontap} />
<svelte:window on:popstate={onpopstate} />

