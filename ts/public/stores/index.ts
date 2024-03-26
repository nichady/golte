import type { Readable, Subscriber } from "svelte/store";
import { state } from "../../shared/appstate.js";

// A svelte store representing the current url.
export const url: Readable<URL> = {
	subscribe(fn: Subscriber<URL>) {
		return state.url.subscribe(fn);
	}
};
