import { RenderError } from "golte/js/shared/renderError.js";

export function isRenderError(err) {
    return err instanceof RenderError;
}
