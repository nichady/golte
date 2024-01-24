import type { ComponentType } from "svelte";

export type ClientNode = {
    comp: string,
    props: Record<string, any>,
    errPage: string,
}

export type ContextData = Record<string, any>;

export type ServerComponent = {
    render(props: Record<string, any>, opts: { context: Map<any, any> }): {
        html: string,
        head: string,
    },

    $$render(
        result: {
            title: string,
            head: string,
            css: Set<string>,
        },
        props: Record<string, any>,
        bindings: unknown,
        slots: unknown,
        context: Map<any, any>,
    ): string,
}

export type CompState = {
    comp: ComponentType,
    props: Record<string, any>,
    errPage: ComponentType,
};
