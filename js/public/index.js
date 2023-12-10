import { getContext } from "svelte"
import { golteContextKey } from "../shared/keys.js";

export function getData() {
    return getContext(golteContextKey);
}