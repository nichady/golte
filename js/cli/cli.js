#!/usr/bin/env node

// @ts-check

import { build } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { cwd } from "node:process";
import { join, relative } from "node:path";
import { copyFile, readFile, mkdir, rename, rm } from "node:fs/promises";
import { existsSync } from "node:fs";
import { build as esbuild } from "esbuild";
import glob from "fast-glob";
import merge from "deepmerge";

// TODO custom log output

/**
 * @typedef {import("../types").Config} Config
 */

async function main() {
    const { templateFile, components, viteConfig, appPath } = await extract(await resolveConfig());

    await buildClient(components, viteConfig, appPath);
    
    const manifest = JSON.parse(await readFile("dist/client/manifest.json", "utf-8"));
    await rm("dist/client/manifest.json")

    await buildServer(components, viteConfig, appPath, manifest);
    await copyFile(templateFile, "dist/server/template.html");
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
 * }>}
 */
async function extract(inputConfig) {
    /** @type {Required<Config>} */
    const defaultConfig = {
        template: "web/app.html",
        srcDir: "web/",
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
    }
}

/**
 * @param {{ name: string, path: string }[]} components
 * @param {import("vite").UserConfig} viteConfig
 * @param {string} appPath 
 */
async function buildClient(components, viteConfig, appPath) {
    /** @type {import("vite").UserConfig} */
    const config = {
        build: {
            ssr: false,
            outDir: "dist/client/",
            minify: false,
            manifest: true,
            // https://github.com/vitejs/vite/issues/4454
            // lib: {},
            rollupOptions: {
                // for some reason, vite sets this to false when using rollupOptions.input instead of lib.entry
                preserveEntrySignatures: "exports-only",
                input: [
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
        manifest += `client: "${component.file}",\n`;
        manifest += `css: [\n`;
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
 */
async function buildServer(components, viteConfig, appPath, manifest) {
    /** @type {import("vite").UserConfig} */
    const config = {
        define: {
            golteImports: await createImports(components),
            golteHydrate: `"${manifest["node_modules/golte/js/client/hydrate.js"].file}"`,
            golteManifest: await createManifest(components, manifest),
        },
        build: {
            ssr: true,
            outDir: "dist/server/",
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
            }
        },
        // appType: "custom",
    };
    
    await build(merge(viteConfig, config));
}

await main();
