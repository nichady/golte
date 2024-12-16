#!/usr/bin/env node

import { build, UserConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { cwd, argv } from "node:process";
import { join, relative, basename, dirname } from "node:path";
import { readFile, rm, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { build as esbuild } from "esbuild";
import glob from "fast-glob";
import merge from "deepmerge";
import replace from '@rollup/plugin-replace';
import { Config } from "../public/config/index.js";
import { embed } from "./templates.js";
import { jsdir, toPosix, clean, traverseCSS } from "./util.js";
import { ClientBuild, ComponentFile, ExtractedConfig, ViteManifest } from "./types.js";
import { pathToFileURL } from "node:url";

async function main() {
    const config = await extract(await resolveConfig());
    await rm(config.outDir, { recursive: true, force: true });

    const client = await buildClient(config);
    await buildServer(config, client);

    if (config.package) writeFile(join(config.outDir, "embed.go"), embed(config.package));
}

async function resolveConfig(): Promise<Config> {
    const defaultConfigFiles = [
        "golte.config.js",
        "golte.config.mjs",
        "golte.config.ts",
        "golte.config.mts",
    ];

    const resolvedPath = defaultConfigFiles.find(existsSync);
    if (!resolvedPath) return {};

    const tempFile = `temp-${Date.now()}.js`;

    await esbuild({
        entryPoints: [resolvedPath],
        outfile: tempFile,
        // write: false,
        platform: "node",
        // bundle: true,
        format: "esm",
        // mainFields: ["main"],
        sourcemap: "inline",
        // metafile: true,
    });

    try {
        return (await import(pathToFileURL(join(cwd(), tempFile)).href)).default;
    } finally {
        await rm(tempFile);
    }
}

async function extract(inputConfig: Config): Promise<ExtractedConfig> {
    const defaultConfig: Required<Config> = {
        template: "web/app.html",
        srcDir: "web/",
        outDir: "build/",
        package: true,
        assets: "golte_",
        mode: "prod",
        vite: {
            build: {
                cssCodeSplit: true,
            },
            plugins: [
                svelte({
                    compilerOptions: {
                        // css: "external",
                        hydratable: true,
                    }
                })
            ]
        },
    }

    const config = merge(defaultConfig, inputConfig);

    let { assets } = config;
    if (assets.startsWith("/")) assets = assets.slice(1);
    if (assets.endsWith("/")) assets = assets.slice(0, -1);

    const paths = await glob(toPosix(join(config.srcDir, "**/*.svelte")));
    const components = paths.map((path) => ({
        name: relative(config.srcDir, path).replace(/\.svelte$/, ""),
        path: path,
    }));

    components.push({ name: "$$$GOLTE_DEFAULT_ERROR$$$", path: `${jsdir}/shared/default-error.svelte` });

    let packageName = "";
    if (config.package === true) {
        packageName = basename(config.outDir);
    } else if (typeof config.package === "string") {
        packageName = config.package;
    }

    let mode = argv[2];
    if (mode !== "dev" && mode !== "prod") mode = config.mode;

    return {
        ...config,
        assets,
        components,
        package: packageName,
        dev: mode === "dev",
    }
}

async function buildClient(config: ExtractedConfig): Promise<ClientBuild> {
    const viteConfig: UserConfig = {
        mode: config.dev ? "development" : "production",
        base: config.assets,
        build: {
            ssr: false,
            outDir: join(config.outDir, "client"),
            manifest: true,
            minify: !config.dev,
            sourcemap: config.dev,
            // lib: {}, // https://github.com/vitejs/vite/issues/4454
            rollupOptions: {
                // for some reason, vite sets this to false when using rollupOptions.input instead of lib.entry
                preserveEntrySignatures: "exports-only",
                input: [
                    `./${config.template}`,
                    `./${jsdir}/client/hydrate.js`,
                    `./${jsdir}/shared/default-error.svelte`,
                    ...config.components.map((c) => `./${c.path}`),
                ],
                output: {
                    format: "es",
                    entryFileNames: "entries/[name]-[hash].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                    assetFileNames: "assets/[name]-[hash].[ext]",
                    sourcemapPathTransform(relativeSourcePath, sourcemapPath) {
                        return pathToFileURL(join(dirname(sourcemapPath), relativeSourcePath)).href;
                    },
                }
            },
        },
        // appType: "custom",
    };

    await build(merge(config.vite, viteConfig));

    const manifestPath = join(config.outDir, "client/.vite/manifest.json");
    const manifestFile = await readFile(manifestPath, "utf-8");
    await rm(manifestPath);
    
    const templatePath = join(config.outDir, "client", config.template);
    const templateFile = await readFile(templatePath);
    await rm(templatePath);
    
    await clean(join(config.outDir, "client"));

    return {
        manifest: JSON.parse(manifestFile),
        template: templateFile,
    };
}

async function createImports(components: ComponentFile[]) {
    let str = ``;
    for (const i in components) {
        const { path } = components[i];
        str += `import component_${i} from "${toPosix(join(cwd(), path))}";\n`
    }
    return str;
}

async function createManifest(components: ComponentFile[], manifest: ViteManifest, base: string) {
    let str = `{\n`;
    for (const i in components) {
        const { name, path } = components[i];
        const component = manifest[path];

        str += `"${toPosix(name)}": {\n`;
        str += `server: component_${i},\n`;
        str += `Client: "${toPosix(join("/", base, component.file))}",\n`;
        str += `CSS: [\n`;
        for (const css of traverseCSS(manifest, component)) {
            str += `"${toPosix(join("/", base, css))}",\n`;
        }
        str += `],\n`;
        str += `},\n`;
    }
    str += `};\n`;

    return str;
}

async function buildServer(config: ExtractedConfig, client: ClientBuild) {
    const viteConfig: UserConfig = {
        plugins: [
            // we can't use define because vite 5 no longer statically replaces
            //@ts-ignore for some reason there is typescript error here
            replace({
                golteImports: await createImports(config.components),
                golteHydrate: `"` + toPosix(join("/", config.assets, client.manifest[jsdir + "/client/hydrate.js"].file)) + `"`,
                golteManifest: await createManifest(config.components, client.manifest, config.assets),
                golteAssets: `"${config.assets}"`,
            })
        ],
        mode: config.dev ? "development" : "production",
        ssr: {
            noExternal: true,
        },
        base: config.assets,
        build: {
            ssr: true,
            outDir: join(config.outDir, "server/"),
            minify: !config.dev,
            sourcemap: config.dev,
            // lib: {}, // https://github.com/vitejs/vite/issues/4454
            rollupOptions: {
                input: [
                    `./${jsdir}/server/render.js`,
                    `./${jsdir}/server/info.js`,
                ],
                output: {
                    format: "cjs",
                    entryFileNames: "[name].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                    assetFileNames: "assets/[name]-[hash].[ext]",
                    sourcemapPathTransform(relativeSourcePath, sourcemapPath) {
                        // using relative path instead
                        // absolute path is broken; for some reason leading "/" is omitted
                        // might have something to do with goja
                        sourcemapPath = relative(cwd(), sourcemapPath);
                        return join(dirname(sourcemapPath), relativeSourcePath);
                    },
                }
            },
        },
        // appType: "custom",
    };

    await build(merge(config.vite, viteConfig));

    await writeFile(join(config.outDir, "/server/template.html"), client.template);
}

await main();

