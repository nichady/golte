import { UserConfig } from "vite";

export type ComponentFile = {
    name: string,
    path: string
};

export type ViteManifest = Record<string, ViteManifestEntry>;

export type ViteManifestEntry = {
    file: string,
    css: string,
    imports: string,
};

export type ExtractedConfig = {
    components: ComponentFile[];
    package: string;
    dev: boolean;

    template: string;
    outDir: string;
    assets: string;
    vite: UserConfig;
};

export type ClientBuild = {
    manifest: ViteManifest;
    template: Buffer;
};
