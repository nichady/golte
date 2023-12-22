import { getContext } from "svelte"
import { golteContext } from "../shared/keys.js";

export function getData() {
    return getContext(golteContext);
}