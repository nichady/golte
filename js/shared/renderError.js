export class RenderError extends Error {

    /**
     * @param { any } err
     * @param { number } index 
     */
    constructor(err, index) {
        // goja doesn't support this constructor
        // super("RenderError: ", { cause: err });

        super(err)
        this.index = index;
    }
}
