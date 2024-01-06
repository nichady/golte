import { getContext } from "svelte"
import { golteContext } from "../shared/keys.js";

/**
 * This store can only be subscribed to during component initialization.
 * 
 * @type {import('svelte/store').Readable<URL>}
 */
export const url = {
	subscribe(fn) {
		return getContext(golteContext).url.subscribe(fn);
	}
};
