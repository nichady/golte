// @ts-check

import Root from "../shared/Root.svelte";
import { RenderError } from "../shared/renderError.js";

// these variables will be set by vite

// @ts-ignore
golteImports;

// @ts-ignore
const hydrate = golteHydrate;

// @ts-ignore
export const manifest = golteManifest;

/**
 * @param {{ comp: string, props: {} }[]} entries
 * @returns {{ head: string, body: string }}
 */
export function render(entries, contextData) {
    const serverNodes = [];
    const clientNodes = [];
    const stylesheets = new Set();

    for (const e of entries) {
        const c = manifest[e.comp];
        serverNodes.push({ comp: c.server, props: e.props });
        clientNodes.push({ comp: `/${c.client}`, props: e.props });
        for (const path of c.css) {
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
        head: head,
        body: html,
    }
}

export function isRenderError(err) {
    return err instanceof RenderError;
}
