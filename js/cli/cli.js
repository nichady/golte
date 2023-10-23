#!/usr/bin/env node

import { build } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { cwd } from "node:process";
import path from "node:path";
import { copyFile, readFile, writeFile, mkdir, rename } from "node:fs/promises";

async function main() {
    const options = (await import(path.join(cwd(), "golte.config.js"))).default;

    await mkdir("dist", { recursive: true });
    await copyFile(options.template, "dist/template.html");

    await buildClient(options);

    await mkdir(".golte/generated", { recursive: true });
    await rename("dist/client/manifest.json", ".golte/generated/clientManifest.json");

    await generateManifest(options);

    await buildServer(options);
}

async function buildClient(options) {
    await build({
        plugins: [svelte({
            emitCss: false,
            compilerOptions: {
                hydratable: true,
            }
        })],
        build: {
            ssr: false,
            outDir: "dist/client/",
            minify: false,
            manifest: true,
            lib: {
                formats: ["es"],
                entry: [
                    ...Object.values(options.components)
                ],
            },
            rollupOptions: {
                output: {
                    entryFileNames: "entries/[name]-[hash].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                }
            },
        },
    });
}

async function generateManifest(options) {
    let manifest = "";

    manifest += `import { Renderer } from "golte/js/ssr.js";\n\n`;
    
    manifest += `const manifest = {`;
    const clientManifest = JSON.parse(await readFile(".golte/generated/clientManifest.json"));
    for (const [name, srcpath] of Object.entries(options.components)) {
        const destpath = clientManifest[srcpath].file;
        manifest += `
    "${name}": {
        server: import("../../${srcpath}"),
        client: "${destpath}",
    },\n`;
    }
    manifest += `};\n`;

    manifest += `
const renderer = new Renderer(manifest);

export function render(components) {
    renderer.render(manifest, components);
}\n`

    await mkdir(".golte/generated", { recursive: true });
    await writeFile(".golte/generated/manifest.js", manifest)
}
                
async function buildServer(options) {
    await build({
        plugins: [svelte({
            emitCss: false,
            compilerOptions: {
                hydratable: true,
            }
        })],
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
                    entryFileNames: "[name].js",
                    chunkFileNames: "chunks/[name]-[hash].js",
                }
            },
        },
    });
}

await main();
