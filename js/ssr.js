// @ts-check

import Root from "./root.svelte"

export class Renderer {
    constructor(manfiest) {
        this.manifest = manfiest;
    }

    render(...components) {
        return Root.render({
            nodes: components.map((c) => this.manifest[c].server),
        });
    }
}
