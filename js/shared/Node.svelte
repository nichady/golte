<!-- Do not import this directly; instead import node-wrapper.js -->

<script>
    import { Node } from "./node-wrapper.js";

	export let node;
    const { next, content } = node;
</script>

<svelte:component this={content.comp} {...content.props}>
	<!-- #key is needed because csr error handling relies on constructor being called again -->
	{#key $next}
		{#if $next}
			<Node node={$next}/>
		{/if}
	{/key}
</svelte:component>