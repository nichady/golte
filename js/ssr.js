import Root from "./root.svelte"

export class Renderer {
    constructor(manfiest) {
        this.manifest = manfiest;
    }

    async render(...components) {
        return Root.render({
            nodes: await Promise.all(components.map(async (c) => (await manifest[c].server).default)),
        });
    }
}
