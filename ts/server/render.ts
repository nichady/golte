import { default as UntypedRoot } from "../shared/Root.svelte";
import { ContextData, ServerComponent } from "../shared/types.js";
import { handleError } from "../shared/keys.js";
import { ErrorProps, ClientNode } from "../shared/types.js";

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

type ServerNode = {
    comp: any,
    props: Record<string, any>,
    errPage: any,
};


type SSRError = {
    index: number,
    props: ErrorProps,
};

export function Render(entries: Entry[], contextData: ContextData, errPage: string) {
    const serverNodes: ServerNode[] = [];
    const clientNodes: ClientNode[] = [];
    const stylesheets = new Set<string>();

    const err = Manifest[errPage];
    if (!err) throw new Error(`"${errPage}" is not a component`);

    for (const e of entries) {
        const c = Manifest[e.Comp];
        if (!c) throw new Error(`"${e.Comp}" is not a component`);
        serverNodes.push({ comp: c.server, props: e.Props, errPage: err.server });
        clientNodes.push({ comp: `${c.Client}`, props: e.Props, errPage: `${err.Client}` });
        for (const path of c.CSS) {
            stylesheets.add(path);
        }
    }

    for (const path of err.CSS) {
        stylesheets.add(path);
    }

    let error: SSRError | undefined;
    const context = new Map(); // TODO dont use context for this
    context.set(handleError, (e: any) => error = e ) 
    let { html, head } = Root.render({ nodes: serverNodes, contextData }, { context });

    for (const path of stylesheets) {
        head += `\n<link href="${path}" rel="stylesheet">`;
    }

    if (error) {
        clientNodes[error.index].ssrError = error.props;
    }

    html += `
        <script>
            (async function () {
                const target = document.currentScript.parentElement;
                const { hydrate } = await import("${hydrate}");
                await hydrate(target, ${stringify(clientNodes)}, ${stringify(contextData)});
            })();
        </script>
    `;

    return {
        Head: head,
        Body: html,
        HasError: !!error,
    }
}

function stringify(object: any) {
    return JSON.stringify(object).replace("</script>", "<\\/script>");
}
