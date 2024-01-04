<div id="page0">
    {#if import.meta.env.SSR}
        <div id="ssr" />
    {:else}
        <div id="csr" />
    {/if}
</div>

<style>
    div {
        font-size: 30px;
    }
</style>
