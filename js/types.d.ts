import type { UserConfig } from "vite";

export type Config = {
    template?: string;
    srcDir?: string,
    staticDir?: string,
    ignore?: string[],
    appPath?: string;
    vite?: UserConfig;
    // TODO option for output dist folder
}
