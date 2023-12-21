// This file is a wrapper for Node.svelte that adds functionality handling errors.
// When an error is thrown during rendering, it catches it and instead
// the specified error page is rendered instead. Works for both ssr and csr.

//@ts-check

import { getContext } from "svelte";
import Original from "./Node.svelte";
import { errorHandle } from "./keys";

const ssrWrapper = {
    $$render: (result, props, bindings, slots, context) => {
        try {
            return Original.$$render(result, props, bindings, slots, context);
        } catch (err) {
            // if there is error, we need to let go code know
            getContext(errorHandle)(err)
            return props.node.content.errPage.$$render(result, { message: err.stack ?? err }, bindings, slots, context);
        }
    }
};

function csrWrapper(options) {
    try {
        return new Original(options);
    } catch (err) {
        return new options.props.node.content.errPage({...options, props: { message: err.stack ?? err }});
    }
};

export const Node = import.meta.env.SSR ? ssrWrapper : csrWrapper;
