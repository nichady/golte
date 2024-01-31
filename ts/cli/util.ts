import { relative, dirname, join, sep, posix } from "node:path";
import { lstat, readdir, rm } from "node:fs/promises";
import { cwd } from "node:process";
import { fileURLToPath } from "node:url";
import { ViteManifest, ViteManifestEntry } from "./types.js";

export const jsdir = toPosix(relative(cwd(), dirname(dirname(fileURLToPath(import.meta.url)))));

export function toPosix(p: string) {
    return p.split(sep).join(posix.sep);
}

/** Recursively removes all empty directories in a given directory */
export async function clean(path: string) {
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

/** Get the css from a manifest entry and its dependencies. */
export function traverseCSS(manifest: ViteManifest, component: ViteManifestEntry) {
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