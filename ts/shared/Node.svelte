<!-- Do not import this directly; instead import node-wrapper.js -->

<script>
	import { Node } from "./node-wrapper.js";

	/**
	 * @typedef {import("./list.js").NodeState<T>} NodeState
	 * @template T
	 */

	/**
	 * @typedef {import("./types.js").CompState} CompState
	 */

	/** @type {NodeState<CompState>} */
	export let node;
	
	/** @type {number} */
	export let index;

	const { next, content } = node;
</script>

<svelte:component this={content.comp} {...content.props}>
	<!-- #key is needed because csr error handling relies on constructor being called again -->
	{#key $next}
		{#if $next}
			<!-- Cannot use svelte:self because need to use wrapper -->
			<Node node={$next} index={index + 1} />
		{/if}
	{/key}
</svelte:component>
