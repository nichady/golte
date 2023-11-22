// This file is a wrapper for Node.svelte that adds functionality dealing with errors during ssr.
// When an error is thrown during rendering, it catches it and instead throws a wrapped error.
// This wrapped error contains the index of the node, which is useful for telling which node is
// responsible for the error when rendering multiple nested layouts.

//@ts-check

import Original from "./Node.svelte";
import { RenderError } from "./renderError.js"

const wrapper = {
    $$render: (result, props, bindings, slots, context) => {
        try {
            return Original.$$render(result, props, bindings, slots, context);
        } catch (err) {
            if (err instanceof RenderError) throw err;
            throw new RenderError(err, props.index);
        }
    }
};

export const Node = import.meta.env.SSR ? wrapper : Original;
