import { default as UntypedRoot } from "../shared/Root.svelte";
import { ContextData, ServerComponent } from "../shared/types.js";
import { markError } from "../shared/keys.js";

const Root: ServerComponent = UntypedRoot as any;

// these variables will be set by vite

// @ts-ignore
golteImports;

// @ts-ignore
const hydrate = golteHydrate;

// @ts-ignore
export const Manifest = golteManifest;

type Entry = {
    Comp: string;
    Props: Record<string, any>;
};

export function Render(entries: Entry[], contextData: ContextData, errPage: string) {
    const serverNodes = [];
    const clientNodes = [];
    const stylesheets = new Set<string>();

    const err = Manifest[errPage];
    if (!err) throw new Error(`"${errPage}" is not a component`);

    for (const e of entries) {
        const c = Manifest[e.Comp];
        if (!c) throw new Error(`"${e.Comp}" is not a component`);
        serverNodes.push({ comp: c.server, props: e.Props, errPage: err.server });
        clientNodes.push({ comp: `/${c.Client}`, props: e.Props, errPage: `/${err.Client}` });
        for (const path of c.CSS) {
            stylesheets.add(path);
        }
    }

    for (const path of err.CSS) {
        stylesheets.add(path);
    }

    let hasError = false;
    const context = new Map();
    context.set(markError, () => hasError = true ) 
    let { html, head } = Root.render({ nodes: serverNodes, contextData }, { context });

    for (const path of stylesheets) {
        head += `\n<link href="/${path}" rel="stylesheet">`;
    }

    html += `
        <script>
            (async function () {
                const target = document.currentScript.parentElement;
                const { hydrate } = await import("/${hydrate}");
                await hydrate(target, ${stringify(clientNodes)}, ${stringify(contextData)});
            })();
        </script>
    `;

    return {
        Head: head,
        Body: html,
        HasError: hasError,
    }
}

function stringify(object: any) {
    return JSON.stringify(object).replace("</script>", "<\\/script>");
}
