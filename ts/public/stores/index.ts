import { getContext } from "svelte"
import { golteContext } from "../../shared/keys.js";
import type { Readable, Subscriber } from "svelte/store";
import type { AppState } from "../../shared/appstate.js";

function getGolteContext() {
	return getContext(golteContext) as AppState;
}

// This store can only be subscribed to during component initialization.
export const url: Readable<URL> = {
	subscribe(fn: Subscriber<URL>) {
		return getGolteContext().url.subscribe(fn);
	}
};
