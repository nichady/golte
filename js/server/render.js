// @ts-check

import Root from "../shared/Root.svelte";
import { RenderError } from "../shared/renderError.js";

// these variables will be set by vite

// @ts-ignore
golteImports;

// @ts-ignore
const hydrate = golteHydrate;

// @ts-ignore
export const Manifest = golteManifest;

/**
 * @param {{ Comp: string, Props: {} }[]} entries
 * @returns {{ Head: string, Body: string }}
 */
export function Render(entries, contextData) {
    const serverNodes = [];
    const clientNodes = [];
    const stylesheets = new Set();

    for (const e of entries) {
        const c = Manifest[e.Comp];
        serverNodes.push({ comp: c.server, props: e.Props });
        clientNodes.push({ comp: `/${c.Client}`, props: e.Props });
        for (const path of c.CSS) {
            stylesheets.add(path);
        }
    }

    let { html, head } = Root.render({ nodes: serverNodes, contextData });

    for (const path of stylesheets) {
        head += `\n<link href="/${path}" rel="stylesheet">`;
    }

    html += `
        <script>
            (async function () {
                const target = document.currentScript.parentElement;
                const { hydrate } = await import("/${hydrate}");
                await hydrate(target, ${JSON.stringify(clientNodes)}, ${JSON.stringify(contextData)});
            })();
        </script>
    `

    return {
        Head: head,
        Body: html,
    }
}

export function IsRenderError(err) {
    return err instanceof RenderError;
}
