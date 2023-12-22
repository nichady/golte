// This file is a wrapper for Node.svelte that adds functionality handling errors.
// When an error is thrown during rendering, it catches it and instead
// the specified error page is rendered instead. Works for both ssr and csr.

//@ts-check

import Original from "./Node.svelte";
import { getContext } from "svelte";
import { errorHandle } from "./keys";

const ssrWrapper = {
    $$render: (result, props, bindings, slots, context) => {
        try {
            return Original.$$render(result, props, bindings, slots, context);
        } catch (err) {
            // if there is error, we need to let render.js know
            const errProps = {
                status: 500,
                message: (err instanceof Error && err.stack) ? err.stack : err.toString(),
            }; 
            getContext(errorHandle)({ props: errProps, index: props.index })
            return props.node.content.errPage.$$render(result, errProps, bindings, slots, context);
        }
    }
};

function csrWrapper(options) {
    // if there as an error during ssr, don't render anything new
    const ssrError = options.props.node.content.ssrError;
    if (ssrError) return new options.props.node.content.errPage({...options, props: ssrError});
    
    try {
        return new Original(options);
    } catch (err) {
        return new options.props.node.content.errPage({...options, props: { message: err.stack ?? err }});
    }
};

export const Node = import.meta.env.SSR ? ssrWrapper : csrWrapper;
