// @ts-check

import Root from "../shared/Root.svelte";
import { markError } from "../shared/keys";

// these variables will be set by vite

// @ts-ignore
golteImports;

// @ts-ignore
const hydrate = golteHydrate;

// @ts-ignore
export const Manifest = golteManifest;

/**
 * @param {{ Comp: string, Props: {} }[]} entries
 * @returns {{ Head: string, Body: string, HasError: boolean }}
 */
export function Render(entries, contextData, errPage) {
    const serverNodes = [];
    const clientNodes = [];
    const stylesheets = new Set();

    const err = Manifest[errPage];

    for (const e of entries) {
        const c = Manifest[e.Comp];
        serverNodes.push({ comp: c.server, props: e.Props, errPage: err.server });
        clientNodes.push({ comp: `/${c.Client}`, props: e.Props, errPage: `/${err.Client}` });
        for (const path of c.CSS) {
            stylesheets.add(path);
        }
    }

    for (const path of err.CSS) {
        stylesheets.add(path);
    }

    let hasError = false;
    const context = new Map();
    context.set(markError, () => hasError = true ) 
    let { html, head } = Root.render({ nodes: serverNodes, contextData }, { context });

    for (const path of stylesheets) {
        head += `\n<link href="/${path}" rel="stylesheet">`;
    }

    html += `
        <script>
            (async function () {
                const target = document.currentScript.parentElement;
                const { hydrate } = await import("/${hydrate}");
                await hydrate(target, ${stringify(clientNodes)}, ${stringify(contextData)});
            })();
        </script>
    `;

    return {
        Head: head,
        Body: html,
        HasError: hasError,
    }
}

function stringify(object) {
    return JSON.stringify(object).replace("</script>", "<\\/script>");
}
