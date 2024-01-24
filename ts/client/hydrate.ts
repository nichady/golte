import Root from "../shared/Root.svelte";
import type { ClientNode, ContextData } from "../shared/types.js"

export async function hydrate(target: HTMLElement, nodes: ClientNode[], contextData: ContextData) {
    const promise = Promise.all(nodes.map(async (n) => ({
        comp: (await import(n.comp)).default,
        props: n.props,
        errPage: (await import(n.errPage)).default,
    })));

    new Root({
        target: target,
        props: {
            nodes: await promise,
            contextData,
        },
        hydrate: true,
    });
}