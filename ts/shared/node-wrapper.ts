// This file is a wrapper for Node.svelte that adds functionality handling errors.
// When an error is thrown during ssr, it catches it and instead
// the specified error page is rendered instead.

import { default as ClientNode } from "./Node.svelte";
import { getContext } from "svelte";
import { handleError } from "./keys.js";
import type { ServerComponent } from "./types.js";

const ServerNode: ServerComponent = ClientNode as any;
const ssrWrapper: ServerComponent = {
    ...ServerNode,
    $$render: (result, props, bindings, slots, context) => {
        try {
            return ServerNode.$$render(result, props, bindings, slots, context);
        } catch (err) {
            let message = "Internal Error";
            if (import.meta.env.MODE === "development") {
                message = (err instanceof Error && err.stack) ? err.stack : String(err);
            }

            const errProps = {
                status: 500,
                message,
            };
            
            getContext<Function>(handleError)({ index: props.index, props: errProps });
            return props.node.content.errPage.$$render(result, errProps, bindings, slots, context);
        }
    }
};

function csrWrapper(options: any): ClientNode {
    // if there as an error during ssr, don't render anything new
    const ssrError = options.props.node.content.ssrError;
    if (ssrError) return new options.props.node.content.errPage({ ...options, props: ssrError });

    return new ClientNode(options);
};

export const Node: typeof ClientNode = import.meta.env.SSR ? ssrWrapper : csrWrapper as any;
