// copies svelte files from ts/ to js/ while keeping directory structure

import glob from "fast-glob";
import { copyFile, mkdir } from "fs/promises";
import { dirname } from "path";

const paths = await glob("**/*.svelte", { cwd: "ts" });

for (const p of paths) {
    await mkdir(dirname("js/" + p), { recursive: true });
    await copyFile("ts/" + p, "js/" + p);
}
