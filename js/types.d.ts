import type { UserConfig } from "vite";

export type Config = {
    template?: string;
    components?: string[];
    vite?: UserConfig;
}
