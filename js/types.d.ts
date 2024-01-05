import type { UserConfig } from "vite";

export type Config = {
    template?: string;
    srcDir?: string,
    outDir?: string;
    ignore?: string[],
    appPath?: string;
    vite?: UserConfig;
}
