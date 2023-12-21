import { writable } from "svelte/store";

/**
 * Creates a singly linked list from the given array, using svelte stores for reactivity
 * @param {any[]} array 
 */
export function fromArray(array) {
    let current = null;
    for (let i = array.length - 1; i >= 0; i--) {
        current = {
            content: array[i],
            next: writable(current),
        }
    }
    return writable(current);
}
