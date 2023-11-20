import Root from "golte/js/root.svelte";

export async function hydrate(target, nodes) {
    nodes = await Promise.all(nodes.map(async (n) => ({
        comp: (await import(n.comp)).default,
        props: n.props,
    })));

    new Root({
        target: target,
        props: { nodes },
        hydrate: true,
    });
}