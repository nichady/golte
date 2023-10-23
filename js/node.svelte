<script>
	export let nodes;
	export let index;

	let Comp = nodes[index];
</script>

<svelte:component this={Comp}>
	{#if nodes[index+1] }
		<svelte:self {nodes} index={index+1} />
	{/if}
</svelte:component>
