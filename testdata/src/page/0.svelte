<div id="page0"/>

{#if import.meta.env.SSR}
    <div id="page0-ssr" />
{:else}
    <div id="page0-csr" />
{/if}

<style>
    div {
        font-size: 30px;
    }
</style>
