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
        const nodes = [];
        const stylesheets = new Set();
        for (const c of components.map((c) => this.manifest[c])) {
            nodes.push(c.server);
            for (const path of c.css) {
                stylesheets.add(path);
            }
        }

        let { html, head } = Root.render({ nodes });

        for (const path of stylesheets) {
            head += `\n<link href="_golte/${path}" rel="stylesheet">`;
        }

        return {
            head: head,
            body: html,
        }
    }
}
