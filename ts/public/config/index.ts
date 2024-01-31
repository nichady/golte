import type { UserConfig } from "vite";

export type Config = {
    /**
     * The template html file. This will be parsed by Go's
     * text/template and given the fields ```.Head``` and ```.Body```
     * 
     * @default "web/app.html"
     */
    template?: string;

    /**
     * The root directory which contains the source svelte files.
     * 
     * Each svelte file in this directory, including svelte files in subdirectories,
     * will be mapped to a component whose name is the full path of the file
     * relative to the ```srcDir``` without the extension.
     * 
     * For example, ```components/page/login.svelte``` will map to the name
     * ```page/login``` if ```srcDir``` is set to "components/"
     * 
     * @default "web/"
     */
    srcDir?: string,

    /**
     * The build output directory. If it already exists,
     * this directory will be removed prior to building.
     * 
     * @default "build/"
     */
    outDir?: string;

    /**
     * Whether or not to directly generate a go package for the directory
     * which will contain code ready to import into other go packages.
     * 
     * If set to false, no package will be generated.
     * If set to true, a package will be generated with the name set to the last portion of outDir.
     * If set to a string, a package will be generated with name set to that string.
     * 
     * @default true
     */
    package?: boolean | string;

    /**
     * The route in which assets such as javascript and css will be served under.
     * 
     * @default "golte_"
     */
    assets?: string;

    /**
     * Whether to build in development mode or production mode.
     * In development mode, sourcemaps will be built, and render error stacktraces will be passed to error components.
     * In production mode, javascript will be minified, and render error stacktraces will not be visible.
     * 
     * This can be overwritten by running "golte dev" or "golte prod".
     * 
     * @default "prod"
     */
    mode?: "dev" | "prod";

    /**
     * Pass additional configuration options to Vite.
     */
    vite?: UserConfig;
};
