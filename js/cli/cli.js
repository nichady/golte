#!/usr/bin/env node

// @ts-check

import { build } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { cwd } from "node:process";
import { join, relative } from "node:path";
import { copyFile, readFile, writeFile, mkdir, rename, unlink } from "node:fs/promises";
import { existsSync } from "node:fs";
import { build as esbuild } from "esbuild";
import glob from "fast-glob";
import merge from "deepmerge";

// TODO custom log output

/**
 * @typedef {import("../types").Config} Config
 */

async function main() {
    const { templateFile, componentMap, viteConfig } = await extract(await resolveConfig());

    await mkdir("dist", { recursive: true });
    await copyFile(templateFile, "dist/template.html");

    await buildClient(componentMap, viteConfig);

    await mkdir(".golte/generated", { recursive: true });
    await rename("dist/client/manifest.json", ".golte/generated/clientManifest.json");

    await generateManifest(componentMap);

    await buildServer(viteConfig);
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
        await unlink(tempFile);
    }
}

/**
 * @param {Config} inputConfig
 * @returns {Promise<{
 *  templateFile: string
 *  componentMap: Record<string, string>
 *  viteConfig: import("vite").UserConfig
 * }>}
 */
async function extract(inputConfig) {
    /** @type {Config} */
    const defaultConfig = {
        template: "web/app.html",
        srcDir: "web/",
        ignore: ["lib/"],
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

    return {
        templateFile: config.template,
        componentMap: Object.fromEntries(
            // (await glob(["**/*.svelte"], { cwd: config.srcDir, ignore: config.ignore }))
            (await glob([join(config.srcDir, "**/*.svelte")], { ignore: config.ignore.map((path) => join(config.srcDir, path)) }))
            .map((path) => [relative(config.srcDir, path).replace(/\.svelte$/, ""), path])
        ),
        viteConfig: config.vite,
    }
}

/**
 * @param {Record<string, string>} componentMap
 * @param {import("vite").UserConfig} viteConfig
 */
async function buildClient(componentMap, viteConfig) {
    /** @type {import("vite").UserConfig} */
    const config = {
        build: {
            ssr: false,
            outDir: "dist/client/",
            minify: false,
            manifest: true,
            lib: {
                formats: ["es"],
                entry: [
                    ...Object.values(componentMap)
                ],
            },
            rollupOptions: {
                output: {
                    entryFileNames: "entries/[name]-[hash].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                    assetFileNames: "assets/[name]-[hash].[ext]",
                }
            },
        },
        // appType: "custom",
    };

    await build(merge(viteConfig, config));
}

/**
 * @param {Record<string, string>} componentMap
 */
async function generateManifest(componentMap) {
    const idxComponentMap = Object.entries(componentMap);

    let manifest = "";

    manifest += `import { Renderer } from "golte/js/ssr.js";\n\n`;

    for (const i in idxComponentMap) {
        const [, srcpath] = idxComponentMap[i];
        manifest += `import component_${i} from "../../${srcpath}";\n`
    }
    manifest += `\n`;

    manifest += `const manifest = {\n`;
    const clientManifest = JSON.parse(await readFile(".golte/generated/clientManifest.json", "utf-8"));
    for (const i in idxComponentMap) {
        const [name, srcpath] = idxComponentMap[i];
        const component = clientManifest[srcpath];

        manifest += `\t"${name}": {\n`;
        manifest += `\t\tserver: component_${i},\n`;
        manifest += `\t\tclient: "${component.file}",\n`;
        manifest += `\t\tcss: [\n`;
        for (const css of component.css ?? []) {
            manifest += `\t\t\t"${css}",\n`;
        }
        manifest += `\t\t],\n`;
        manifest += `\t},\n`;



    }
    manifest += `};\n`;

    manifest += `
const renderer = new Renderer(manifest);

export function render(components) {
    return renderer.render(components);
}\n`

    await mkdir(".golte/generated", { recursive: true });
    await writeFile(".golte/generated/manifest.js", manifest)
}

/**
 * @param {import("vite").UserConfig} viteConfig
 */
async function buildServer(viteConfig) {
    /** @type {import("vite").UserConfig} */
    const config = {
        build: {
            ssr: true,
            outDir: "dist/server/",
            minify: false,
            lib: {
                formats: ["cjs"],
                entry: {
                    "manifest": ".golte/generated/manifest.js",
                },
            },
            rollupOptions: {
                output: {
                    inlineDynamicImports: true,
                    entryFileNames: "[name].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                }
            },
        },
        // appType: "custom",
    };
    
    await build(merge(viteConfig, config));
}

await main();
