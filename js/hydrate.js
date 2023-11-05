import Root from "golte/js/root.svelte";

export async function hydrate(target, nodes) {
    new Root({
        target: target,
        props: {
            nodes: await Promise.all(nodes.map(async (n) => (await import("/_golte/" + n)).default)),
        },
        hydrate: true,
    });
}