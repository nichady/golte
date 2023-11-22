export class RenderError extends Error {
    constructor(err, index) {
        // goja doesn't support this constructor
        // super("Render Error: ", { cause: err });

        super("RenderError")
        this.cause = err;
        this.index = index;
    }
}

export function isRenderError(err) {
    return err instanceof RenderError;
}
