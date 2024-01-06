import { Config } from "../js/public/config";

export default {
    template: "src/app.html",
    srcDir: "src/",
    outDir: "dist/",
    ignore: ["lib/"],
    appPath: "app_",
} satisfies Config;
