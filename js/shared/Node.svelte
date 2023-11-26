<!-- Do not use this component directly; instead use wrapper file. -->

<script>
	import { Node } from "./node-wrapper.js";

	export let nodes;
	export let index;

	$: ({ comp, props } = nodes[index]);
</script>

<svelte:component this={comp} {...props}>
	{#if nodes[index+1] }
		<!-- do not use <svelte:self>, we need to use wrapper -->
		<Node {nodes} index={index+1} />
	{/if}
</svelte:component>
