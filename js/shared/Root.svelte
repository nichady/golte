<script>
    import { Node } from "./node-wrapper.js";

    export let nodes;

    const golteData = Symbol("golteData");

    async function onclick(e) {
        if (!(e.target instanceof HTMLAnchorElement)) return;
        if (e.target.origin !== location.origin) return;
        if (!e.target.hasAttribute("noreload")) return;

        e.preventDefault();
        const json = e.target[golteData] ?? await load(e.target.href);
        await update(json);
        history.pushState({ golte: json }, "", e.target.href);
    }

    async function onhover(e) {
        if (!(e.target instanceof HTMLAnchorElement)) return;
        if (e.target.origin !== location.origin) return;
        if (e.target[golteData]) return;
        if (e.target.getAttribute("noreload") !== "hover") return;

        const json = await load(e.target.href);
        e.target[golteData] = json;
    }

    async function ontap(e) {
        if (!(e.target instanceof HTMLAnchorElement)) return;
        if (e.target.origin !== location.origin) return;
        if (e.target[golteData]) return;
        if (e.target.getAttribute("noreload") !== "tap") return;

        const json = await load(e.target.href);
        e.target[golteData] = json;
    }

    async function onpopstate(e) {
        if (!e.state || !e.state.golte) return
        update(e.state.golte);
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
    async function update(json) {
        const promises = json.map(async (entry) => ({
            comp: (await import(entry.File)).default,
            props: entry.Props,
        }));

        nodes = await Promise.all(promises);
    }
</script>

{#if nodes[0]}
    <Node nodes={nodes} index={0} />
{/if}

<svelte:document on:click={onclick} on:mouseover={onhover} on:mousedown={ontap} on:touchstart={ontap} />
<svelte:window on:popstate={onpopstate} />

