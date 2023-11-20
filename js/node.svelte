<script>
	export let nodes;
	export let index;

	const { comp, props } = nodes[index];
</script>

<svelte:component this={comp} {...props}>
	{#if nodes[index+1] }
		<svelte:self {nodes} index={index+1} />
	{/if}
</svelte:component>
