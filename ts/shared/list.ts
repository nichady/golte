import { writable, Writable } from "svelte/store";

// Creates a singly linked list from the given array, using svelte stores for reactivity
export function fromArray<T>(array: T[]) {
    let current: NodeState<T> | null = null;
    for (let i = array.length - 1; i >= 0; i--) {
        current = {
            content: array[i],
            next: writable(current),
        }
    }
    return writable(current);
}

export type NodeState<T> = {
    content: T,
    next: Writable<NodeState<T> | null>,
};
