<script>
    import { Node } from "./node-wrapper.js";

    export let nodes = [];
</script>

{#if nodes.length > 0}
    <Node nodes={nodes} index={0} />
{/if}
