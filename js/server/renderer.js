// @ts-check

import Root from "golte/js/shared/Root.svelte";

export class Renderer {
    constructor(manfiest) {
        this.manifest = manfiest;
    }

    /**
     * @param {string} hydrate
     * @param {{ comp: string, props: {} }[]} entries
     */
    render(hydrate, entries) {
        const serverNodes = [];
        const clientNodes = [];
        const stylesheets = new Set();

        for (const e of entries) {
            const c = this.manifest[e.comp];
            serverNodes.push({ comp: c.server, props: e.props });
            clientNodes.push({ comp: `/${c.client}`, props: e.props });
            for (const path of c.css) {
                stylesheets.add(path);
            }
        }

        let { html, head } = Root.render({ nodes: serverNodes });

        for (const path of stylesheets) {
            head += `\n<link href="/${path}" rel="stylesheet">`;
        }

        html += `
            <script>
                (async function () {
                    const target = document.currentScript.parentElement;
                    const { hydrate } = await import("/${hydrate}");
                    await hydrate(target, ${JSON.stringify(clientNodes)});
                })();
            </script>
        `

        return {
            head: head,
            body: html,
        }
    }
}
