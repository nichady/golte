import { writable, Writable } from "svelte/store";

// Creates a singly linked list from the given array, using svelte stores for reactivity
export function fromArray<T>(array: T[]): StoreList<T> {
    let current: StoreList<T> = writable(null);
    for (let i = array.length - 1; i >= 0; i--) {
        current = writable({
            content: array[i],
            next: current,
        });
    }
    return current;
}

export type StoreList<T> = Writable<null | {
    content: T,
    next: StoreList<T>,
}>
