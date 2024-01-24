// This file is a wrapper for Node.svelte that adds functionality handling errors.
// When an error is thrown during ssr, it catches it and instead
// the specified error page is rendered instead.

import { default as ClientNode } from "./Node.svelte";
import { getContext } from "svelte";
import { markError } from "./keys.js";
import type { ServerComponent } from "./types.js";

const ServerNode: ServerComponent = ClientNode as any;
const ssrWrapper: ServerComponent = {
    ...ServerNode,
    $$render: (result, props, bindings, slots, context) => {
        try {
            return ServerNode.$$render(result, props, bindings, slots, context);
        } catch (err: any) {
            getContext<Function>(markError)(); // if there is error, we need to let render.js know
            return props.node.content.errPage.$$render(result, {
                status: 500,
                message: (err instanceof Error && err.stack) ? err.stack : err.toString(),
            }, bindings, slots, context);
        }
    }
};

export const Node: typeof ClientNode = import.meta.env.SSR ? ssrWrapper : ServerNode as any;
