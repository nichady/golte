// @ts-check

import Root from "./root.svelte"

export class Renderer {
    constructor(manfiest) {
        this.manifest = manfiest;
    }

    /**
     * @param {string[]} components
     */
    render(components) {
        const serverNodes = [];
        const clientNodes = [];
        const stylesheets = new Set();
        for (const c of components.map((c) => this.manifest[c])) {
            serverNodes.push(c.server);
            clientNodes.push(c.client);
            for (const path of c.css) {
                stylesheets.add(path);
            }
        }

        let { html, head } = Root.render({ nodes: serverNodes });

        for (const path of stylesheets) {
            head += `\n<link href="_golte/${path}" rel="stylesheet">`;
        }

        html += `
            <script>
                (async function () {
                    const target = document.currentScript.parentElement;
                    const { hydrate } = await import("/_golte/hydrate.js");
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
