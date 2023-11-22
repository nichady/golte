<!-- Do not use this component directly; instead use the wrapper file. -->

<script>
	import { Node } from "./node.js";

	export let nodes;
	export let index;

	const { comp, props } = nodes[index];
</script>

<svelte:component this={comp} {...props}>
	{#if nodes[index+1] }
		<!-- do not use <svelte:self>, we need to use wrapper -->
		<Node {nodes} index={index+1} />
	{/if}
</svelte:component>
