// @ts-check

import Root from "../shared/Root.svelte";

export async function hydrate(target, nodes, contextData) {
    const promise = Promise.all(nodes.map(async (n) => ({
        comp: (await import(n.comp)).default,
        props: n.props,
        errPage: (await import(n.errPage)).default,
    })));

    new Root({
        target: target,
        props: {
            nodes: await promise,
            promise,
            contextData,
        },
        hydrate: true,
    });
}