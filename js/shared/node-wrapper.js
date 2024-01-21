// This file is a wrapper for Node.svelte that adds functionality handling errors.
// When an error is thrown during rendering, it catches it and instead
// the specified error page is rendered instead. Works for both ssr and csr.

//@ts-check

import Original from "./Node.svelte";
import { getContext } from "svelte";
import { markError } from "./keys";

const ssrWrapper = {
    $$render: (result, props, bindings, slots, context) => {
        try {
            return Original.$$render(result, props, bindings, slots, context);
        } catch (err) {
            getContext(markError)(); // if there is error, we need to let render.js know
            return props.node.content.errPage.$$render(result, {
                status: 500,
                message: (err instanceof Error && err.stack) ? err.stack : err.toString(),
            }, bindings, slots, context);
        }
    }
};

export const Node = import.meta.env.SSR ? ssrWrapper : Original;
