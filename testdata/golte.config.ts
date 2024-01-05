import { Config } from "../js/types";

export default {
    template: "src/app.html",
    srcDir: "src/",
    outDir: "dist/",
    ignore: ["lib/"],
    appPath: "app_",
} satisfies Config;
