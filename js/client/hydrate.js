// @ts-check

import Root from "../shared/Root.svelte";

export async function hydrate(target, nodes, contextData) {
    nodes = await Promise.all(nodes.map(async (n) => ({
        comp: (await import(n.comp)).default,
        props: n.props,
        errPage: (await import(n.errPage)).default,
        ssrError: n.ssrError,
    })));

    new Root({
        target: target,
        props: { nodes, contextData },
        hydrate: true,
    });
}