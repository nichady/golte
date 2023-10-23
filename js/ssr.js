import Root from "./root.svelte"

export function render(...components) {
    const result = Root.render({
        nodes: components,
    });

    return result;
}

export async function render2(manifest, ...components) {
    const result = Root.render({
        nodes: await Promise.all(components.map(async (c) => (await manifest[c].server).default)),
    });

    return result;
}
