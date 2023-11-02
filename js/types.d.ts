import type { UserConfig } from "vite";

export type Config = {
    template?: string;
    srcDir?: string,
    ignore?: string[],
    vite?: UserConfig;
    // TODO option for output dist folder
}
