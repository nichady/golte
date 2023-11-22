// This file is a wrapper for Node.svelte that adds functionality dealing with errors during ssr.
// When an error is thrown during rendering, it catches it and instead throws a wrapped error.
// This wrapped error contains the index of the node, which is useful for telling which node is
// responsible for the error when rendering multiple nested layouts.

//@ts-check

import Node from "./node.svelte";
import { RenderError } from "./exports.js"

const wrapper = { };

if (import.meta.env.SSR) {
    wrapper.$$render = (result, props, bindings, slots, context) => {
        try {
            return Node.$$render(result, props, bindings, slots, context);
        } catch (err) {
            throw new RenderError(err, props.index);
        }
    }
}

export { wrapper as Node };
