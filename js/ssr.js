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
        return Root.render({
            nodes: components.map((c) => this.manifest[c].server),
        });
    }
}
