import { get, Writable, writable } from "svelte/store";
import { fromArray, StoreList } from "./list.js";
import { CompState } from "./types.js";

type CSRResponse = {
    Entries: ResponseEntry[],
    ErrPage: ResponseEntry,
}

type ResponseEntry = {
    File: string,
    Props: Record<string, any>,
    CSS: string[],
}

export let state: {
    url: Writable<URL>;
    node: StoreList<CompState>;

    // client only properties below
    hrefMap: Record<string, Promise<CompState[]>>;
    update: (href: string) => Promise<void>;
};

export function initState(url: string, nodes: CompState[]) {
    state = {} as typeof state;
    
    state.url = writable(new URL(url));
    state.node = fromArray(nodes);

    if (!import.meta.env.SSR) {
        state.hrefMap = {
            [url]: new Promise(r => r(nodes)),
        };
    
        state.update = async (href: string) => {
            state.url.set(new URL(href))
        
            const array = await (state.hrefMap[href] ?? load(href));
        
            // this loop replaces the first differentiated node from after onto before
            // the reason this is done instead of simply replacing the first node is so we don't rerender unnecessary nodes
            // this allows for data persistence in already rendered nodes
            let before = state.node;
            let after = fromArray(array);
            while (true) {
                const bval = get(before);
                const aval = get(after);
        
                if (!bval && !aval) break; // both nodes are null - end of list, no diff
        
                const bcomp = bval?.content.comp;
                const acomp = aval?.content.comp;
        
                if (bcomp === acomp) { // nodes are same component - pass
                    // neiter bval nor aval can be null at this point - typescript isn't smart enough to figure that out
        
                    //@ts-ignore
                    before = bval.next;
        
                    //@ts-ignore
                    after = aval.next;
                } else { // nodes are different components - replace
                    before.set(aval);
                    break;
                }
            }
        }
    }
}

// export const AppState: typeof ClientAppState = import.meta.env.SSR ? ServerAppState : ClientAppState as any;
// export type AppState = ClientAppState;

export async function load(href: string) {
    const headers = { "Golte": "true" };
    const resp = await fetch(href, { headers });
    const json: CSRResponse = await resp.json();

    for (const entry of [...json.Entries, json.ErrPage]) {
        // load css
        for (const css of entry.CSS) {
            if (document.querySelector(`link[href="${css}"][rel="stylesheet"]`)) continue;
            const link = document.createElement("link");
            link.href = css;
            link.rel = "stylesheet";
            document.head.appendChild(link);
        }
        // TODO send css as its own field, outside of the array
    }

    const promises = json.Entries.map(async (entry) => ({
        comp: (await import(entry.File)).default,
        props: entry.Props,
        errPage: (await import(json.ErrPage.File)).default,
    }));

    return await Promise.all(promises);
}
