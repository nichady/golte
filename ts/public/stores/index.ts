import { getContext } from "svelte"
import { golteContext } from "../../shared/keys.js";
import type { Readable, Subscriber } from "svelte/store";
import { GolteContext } from "../../shared/types.js";

function getGolteContext() {
	return getContext(golteContext) as GolteContext;
}

// This store can only be subscribed to during component initialization.
export const url: Readable<URL> = {
	subscribe(fn: Subscriber<URL>) {
		return getGolteContext().url.subscribe(fn);
	}
};
