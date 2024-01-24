#!/usr/bin/env node

import { build, UserConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { cwd } from "node:process";
import path, { join, relative, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { readFile, rename, rm, readdir, lstat } from "node:fs/promises";
import { existsSync } from "node:fs";
import { build as esbuild } from "esbuild";
import glob from "fast-glob";
import merge from "deepmerge";
import replace from '@rollup/plugin-replace';
import { Config } from "../public/config/index.js";

type Component = {
    name: string,
    path: string
};

type ViteManifest = Record<string, ViteManifestEntry>;

type ViteManifestEntry = {
    file: string,
    css: string,
    imports: string,
};

const jsdir = relative(cwd(), dirname(dirname(fileURLToPath(import.meta.url))));

function toPosix(p: string) {
    return p.split(path.sep).join(path.posix.sep);
}

async function main() {
    const { templateFile, components, viteConfig, appPath, outDir } = await extract(await resolveConfig());

    await buildClient(components, viteConfig, appPath, templateFile, outDir);

    const manifest = JSON.parse(await readFile(join(outDir, "client/.vite/manifest.json"), "utf-8"));
    await rm(join(outDir, "/client/.vite/"), { recursive: true });

    await buildServer(components, viteConfig, appPath, manifest, outDir);

    await rename(join(outDir, "/client", templateFile), join(outDir, "/server/template.html"));
    await clean(join(outDir, "/client"));
}

/**false
 * @returns {Promise<Config>}
 */
async function resolveConfig() {
    const defaultConfigFiles = [
        "golte.config.js",
        "golte.config.mjs",
        "golte.config.ts",
        "golte.config.mts",
    ];

    let resolvedPath = "";
    for (const filename of defaultConfigFiles) {
        const filepath = join(cwd(), filename);
        if (!existsSync(filepath)) continue;
        resolvedPath = filepath;
        break;
    }

    if (!resolvedPath) return {};

    const tempFile = `temp-${Date.now()}.js`;

    await esbuild({
        absWorkingDir: cwd(),
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
        // file:// is necessary for windows
        /** @type {Config} */
        const configFile = (await import("file://" + join(cwd(), tempFile))).default
        return configFile;
    } finally {
        await rm(tempFile);
    }
}

async function extract(inputConfig: Config) {
    const defaultConfig: Required<Config> = {
        template: "web/app.html",
        srcDir: "web/",
        outDir: "dist/",
        ignore: ["lib/"],
        appPath: "_app",
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

    if (config.appPath.startsWith("/")) config.appPath = config.appPath.slice(1);
    if (config.appPath.endsWith("/")) config.appPath = config.appPath.slice(0, -1);

    const ignore = config.ignore.map((path) => join(config.srcDir, path));
    const paths = await glob(toPosix(join(config.srcDir, "**/*.svelte")), { ignore });
    const components = paths.map((path) => ({
        name: relative(config.srcDir, path).replace(/\.svelte$/, ""),
        path: path,
    }));

    components.push({ name: "$$$GOLTE_DEFAULT_ERROR$$$", path: `${jsdir}/shared/default-error.svelte` });

    return {
        templateFile: config.template,
        components: components,
        viteConfig: config.vite,
        appPath: config.appPath,
        outDir: config.outDir,
    }
}

async function buildClient(
    components: Component[],
    viteConfig: UserConfig,
    appPath: string,
    templateFile: string,
    outDir: string) {
    const config: UserConfig = {
        build: {
            ssr: false,
            outDir: join(outDir, "client"),
            minify: false,
            manifest: true,
            // https://github.com/vitejs/vite/issues/4454
            // lib: {},
            rollupOptions: {
                // for some reason, vite sets this to false when using rollupOptions.input instead of lib.entry
                preserveEntrySignatures: "exports-only",
                input: [
                    `./${templateFile}`,
                    `./${jsdir}/client/hydrate.js`,
                    `./${jsdir}/shared/default-error.svelte`,
                    ...components.map((c) => `./${c.path}`),
                ],
                output: {
                    format: "es",
                    entryFileNames: `${appPath}/entries/[name]-[hash].js`,
                    chunkFileNames: `${appPath}/chunks/[name]-[hash].js`,
                    assetFileNames: `${appPath}/assets/[name]-[hash].[ext]`,
                }
            },
        },
        // appType: "custom",
    };

    await build(merge(viteConfig, config));
}

async function createImports(components: Component[]) {
    let imports = ``;
    for (const i in components) {
        const { path } = components[i];
        imports += `import component_${i} from "${toPosix(join(cwd(), path))}";\n`
    }
    return imports;
}

async function createManifest(components: Component[], manifestFile: ViteManifest) {
    let manifest = `{\n`;
    for (const i in components) {
        const { name, path } = components[i];
        const component = manifestFile[path];

        manifest += `"${toPosix(name)}": {\n`;
        manifest += `server: component_${i},\n`;
        manifest += `Client: "${component.file}",\n`;
        manifest += `CSS: [\n`;
        for (const css of traverseCSS(manifestFile, component)) {
            manifest += `"${css}",\n`;
        }
        manifest += `],\n`;
        manifest += `},\n`;
    }
    manifest += `};\n`;

    return manifest;
}

function traverseCSS(manifest: ViteManifest, component: ViteManifestEntry) {
    const css = new Set(component.css);

    for (const i of component.imports ?? []) {
        if (!(i in manifest)) continue;
        const component = manifest[i];
        for (const c of traverseCSS(manifest, component)) {
            css.add(c);
        }
    }

    return css;
}

async function buildServer(
    components: Component[],
    viteConfig: UserConfig,
    appPath: string,
    manifest: any,
    outDir: string,
) {
    const config: UserConfig = {
        plugins: [
            // we can't use define because vite 5 no longer statically replaces
            //@ts-ignore for some reason there is typescript error here
            replace({
                golteImports: await createImports(components),
                golteHydrate: '"' + manifest[`${jsdir}/client/hydrate.js`].file + '"',
                golteManifest: await createManifest(components, manifest),
            })
        ],
        build: {
            ssr: true,
            outDir: join(outDir, "server/"),
            minify: false,
            // https://github.com/vitejs/vite/issues/4454
            // lib: {},
            rollupOptions: {
                input: [
                    `./${jsdir}/server/render.js`,
                ],
                output: {
                    format: "cjs",
                    entryFileNames: "[name].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                    assetFileNames: `${appPath}/assets/[name]-[hash].[ext]`,
                }
            },
        },
        // appType: "custom",
    };
    
    await build(merge(viteConfig, config));
}

// Removes all empty directories in a given directory
async function clean(path: string) {
    for (let item of await readdir(path)) {
        item = join(path, item);
        const stat = await lstat(item);
        if (!stat.isDirectory()) continue;

        await clean(item);
        if ((await readdir(item)).length === 0) {
            rm(item, { recursive: true });
        };
    }
}

await main();
