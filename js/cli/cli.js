#!/usr/bin/env node

// @ts-check

import { build } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { cwd } from "node:process";
import { join, relative } from "node:path";
import { readFile, rename, rm, readdir, lstat, cp } from "node:fs/promises";
import { existsSync } from "node:fs";
import { build as esbuild } from "esbuild";
import glob from "fast-glob";
import merge from "deepmerge";
import replace from '@rollup/plugin-replace';

// TODO custom log output

/**
 * @typedef {import("../types").Config} Config
 */

async function main() {
    const { templateFile, components, viteConfig, appPath, staticDir, outDir } = await extract(await resolveConfig());

    await buildClient(components, viteConfig, appPath, templateFile, outDir);
    await cp(staticDir, join(outDir, "client/", appPath), { recursive: true });

    const manifest = JSON.parse(await readFile(join(outDir, "client/.vite/manifest.json"), "utf-8"));
    await rm(join(outDir, "/client/.vite/"), { recursive: true });

    await buildServer(components, viteConfig, appPath, manifest, outDir);

    await rename(join(outDir, "/client", templateFile), join(outDir, "/server/template.html"));
    await clean(join(outDir, "/client"));
}

/**
 * @returns {Promise<Config>}
 */
async function resolveConfig() {
    const defaultConfigFiles = [
        "golte.config.js",
        "golte.config.mjs",
        "golte.config.ts",
        "golte.config.mts",
    ];

    let resolvedPath = undefined;
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
        /** @type {Config} */
        const configFile = (await import(join(cwd(), tempFile))).default
        return configFile;
    } finally {
        await rm(tempFile);
    }
}

/**
 * @param {Config} inputConfig
 * @returns {Promise<{
 *  templateFile: string
 *  components: { name: string, path: string }[]
 *  viteConfig: import("vite").UserConfig
 *  appPath: string
 *  staticDir: string
 *  outDir: string
 * }>}
 */
async function extract(inputConfig) {
    /** @type {Required<Config>} */
    const defaultConfig = {
        template: "web/app.html",
        srcDir: "web/",
        outDir: "dist/",
        staticDir: "web/static/",
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
    const paths = await glob([join(config.srcDir, "**/*.svelte")], { ignore });
    const components = paths.map((path) => ({
        name: relative(config.srcDir, path).replace(/\.svelte$/, ""),
        path: path,
    }));

    return {
        templateFile: config.template,
        components: components,
        viteConfig: config.vite,
        appPath: config.appPath,
        staticDir: config.staticDir,
        outDir: config.outDir,
    }
}

/**
 * @param {{ name: string, path: string }[]} components
 * @param {import("vite").UserConfig} viteConfig
 * @param {string} appPath
 * @param {string} templateFile 
 * @param {string} outDir
 */
async function buildClient(components, viteConfig, appPath, templateFile, outDir) {
    /** @type {import("vite").UserConfig} */
    const config = {
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
                    templateFile,
                    "node_modules/golte/js/client/hydrate.js",
                    ...components.map((c) => c.path),
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

/**
 * @param {{ name: string, path: string }[]} components 
 * @returns string
 */
async function createImports(components) {
    let imports = ``;
    for (const i in components) {
        const { path } = components[i];
        imports += `import component_${i} from "${join(cwd(), path)}";\n`
    }
    return imports;
}

/**
 * @param {{ name: string, path: string }[]} components 
 * @param {any} manifestFile
 * @returns string
 */
async function createManifest(components, manifestFile) {
    let manifest = `{\n`;
    for (const i in components) {
        const { name, path } = components[i];
        const component = manifestFile[path];

        manifest += `"${name}": {\n`;
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

function traverseCSS(manifest, component) {
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

/**
 * @param components {{ name: string, path: string }[]}
 * @param {import("vite").UserConfig} viteConfig
 * @param {string} appPath
 * @param {any} manifest
 * @param {string} outDir
 */
async function buildServer(components, viteConfig, appPath, manifest, outDir) {
    /** @type {import("vite").UserConfig} */
    const config = {
        plugins: [
            // we can't use define because vite 5 no longer statically replaces
            replace({
                golteImports: await createImports(components),
                golteHydrate: `"${manifest["node_modules/golte/js/client/hydrate.js"].file}"`,
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
                    "node_modules/golte/js/server/render.js",
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

/**
 * Removes all empty directories in a given directory
 * @param {string} path
 */
async function clean(path) {
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
