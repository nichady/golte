import { Config } from "../js/types";

export default {
    template: "src/app.html",
    srcDir: "src/",
    outDir: "dist/",
    staticDir: "src/static/",
    ignore: ["lib/"],
    appPath: "app_",
} satisfies Config;
