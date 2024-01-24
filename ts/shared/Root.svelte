<script>
    import { readonly, writable } from "svelte/store";
    import { Node } from "./node-wrapper.js";
    import { onMount, setContext } from "svelte";
    import { golteContext} from "./keys.js"
    import { fromArray } from "./list.js"
    import { get } from "svelte/store"

    /** @typedef {import("./types.js").CompState} CompState */
    /** @typedef {import("./types.js").ContextData} ContextData */

    /** @type CompState[] */
    export let nodes;

    /** @type Promise<CompState[]> */
    export let promise;

    /** @type ContextData */
    export let contextData;

    const node = fromArray(nodes);

    const url = writable(new URL(contextData.URL));
    setContext(golteContext, {
        url: readonly(url),
    });

    const hrefMap = {
        [$url.href]: promise,
    }

    onMount(() => {
        history.replaceState($url.href, "");

        /** @type {(this: HTMLAnchorElement) => Promise<void>} */ 
        async function on() {
            if (this.href in hrefMap) return;
            hrefMap[this.href] = load(this.href);
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
                await update(a.href);
                history.pushState(a.href, "", a.href);
            })
        }

        window.addEventListener("popstate", async (e) => {
            if (!e.state) return;
            await update(e.state);
        });
    });

    /**
     * @param {string} href
     */
    async function load(href) {
        /**
         * @typedef ResponseEntry
         * @prop {string} File
         * @prop {Record<string, any>} Props
         * @prop {string[]} CSS
         */

        /**
         * @typedef CSRResponse
         * @prop {ResponseEntry[]} Entries
         * @prop {ResponseEntry} ErrPage
         */

        const headers = {
            "Golte": "true",
        };

        const resp = await fetch(href, { headers });

        /** @type {CSRResponse} */
        const json = await resp.json();

        for (const entry of [...json.Entries, json.ErrPage]) {
            // load css
            for (const css of entry.CSS) {
                if (document.querySelector(`link[href="${css}"][rel="stylesheet"]`)) continue;
                const link = document.createElement("link");
                link.href = css;
                link.rel = "stylesheet";
                document.head.appendChild(link);
            }
            // TODO send css as its own field, outside of the array
        }

        const promises = json.Entries.map(async (entry) => ({
            comp: (await import(entry.File)).default,
            props: entry.Props,
            errPage: (await import(json.ErrPage.File)).default,
        }));

        return await Promise.all(promises);
    }

    /**
     * @param {string} href
     */
    async function update(href) {
        $url = new URL(href);

        const array = await (hrefMap[href] ?? load(href));

        // this loop replaces the first differentiated node from after onto before
        // the reason this is done instead of simply replacing the first node is so we don't rerender unnecessary nodes
        // this allows for data persistence in already rendered nodes
        let before = node;
        let after = fromArray(array);
        while (true) {
            const bval = get(before);
            const aval = get(after);

            if (!bval && !aval) break; // both nodes are null - end of list, no diff

            const bcomp = bval?.content.comp;
            const acomp = aval?.content.comp;

            if (bcomp === acomp) { // nodes are same component - pass
                // neiter bval nor aval can be null at this point - typescript isn't smart enough to figure that out
                
                //@ts-ignore
                before = bval.next;
                
                //@ts-ignore
                after = aval.next;
            } else { // nodes are different components - replace
                before.set(aval);
                break;
            }
        }
    }
</script>

<!-- #key is needed because csr error handling relies on constructor being called again -->
{#key $node}
    {#if $node}
        <Node node={$node} index={0} />
    {/if}
{/key}
