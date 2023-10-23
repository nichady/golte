#!/usr/bin/env node

import { build } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

import { globSync } from "glob";

const options = {
    template: "web/app.html",
    components: Object.fromEntries(
        globSync("web/**/*.svelte", { ignore: "web/lib/**/*.svelte" })
        .map((path) => [path.replace(/^web\//, "").replace(/\.svelte$/, ""), path])
    ),
}

build({
    plugins: [svelte({
        emitCss: false,
        compilerOptions: {
            hydratable: true,
        }
    })],
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
});
