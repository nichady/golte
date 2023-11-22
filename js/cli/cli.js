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

    await mkdir(".golte/generated", { recursive: true });
    await copyFile("node_modules/golte/js/client/hydrate.js", ".golte/generated/hydrate.js"),

    await buildClient(componentMap, viteConfig);
    await rename("dist/client/manifest.json", ".golte/generated/clientManifest.json");

    await generateRenderfile(componentMap);
    await buildServer(viteConfig);
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
                    ".golte/generated/hydrate.js",
                    ...Object.values(componentMap),
                ],
            },
            rollupOptions: {
                output: {
                    entryFileNames: (chunk) => {
                        if (relative(cwd(), chunk.facadeModuleId ?? "") === ".golte/generated/hydrate.js") {
                            return "js/hydrate.js";
                        }
                        return "js/[name]-[hash].js"
                    },
                    chunkFileNames: "js/[name]-[hash].js",
                    assetFileNames: "[ext]/[name]-[hash].[ext]",
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
async function generateRenderfile(componentMap) {
    const idxComponentMap = Object.entries(componentMap);

    let renderfile = "";

    renderfile += `import { Renderer } from "golte/js/server";\n\n`;

    for (const i in idxComponentMap) {
        const [, srcpath] = idxComponentMap[i];
        renderfile += `import component_${i} from "../../${srcpath}";\n`
    }
    renderfile += `\n`;

    renderfile += `const manifest = {\n`;
    const clientManifest = JSON.parse(await readFile(".golte/generated/clientManifest.json", "utf-8"));
    for (const i in idxComponentMap) {
        const [name, srcpath] = idxComponentMap[i];
        const component = clientManifest[srcpath];

        renderfile += `\t"${name}": {\n`;
        renderfile += `\t\tserver: component_${i},\n`;
        renderfile += `\t\tclient: "${component.file}",\n`;
        renderfile += `\t\tcss: [\n`;
        for (const css of component.css ?? []) {
            renderfile += `\t\t\t"${css}",\n`;
        }
        renderfile += `\t\t],\n`;
        renderfile += `\t},\n`;



    }
    renderfile += `};\n`;

    renderfile += `
const renderer = new Renderer(manifest);

export function render(assetsPath, components) {
    return renderer.render(assetsPath, components);
}\n`

    await mkdir(".golte/generated", { recursive: true });
    await writeFile(".golte/generated/renderfile.js", renderfile)
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
                entry: [".golte/generated/renderfile.js", "node_modules/golte/js/server/exports.js"],
            },
        },
        // appType: "custom",
    };
    
    await build(merge(viteConfig, config));
}

await main();
