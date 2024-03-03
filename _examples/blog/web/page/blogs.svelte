<script>
    import { preload } from "golte";

    export let blogs;
</script>

{#if blogs.length === 0}
    No blog posts yet.
{:else}
    <div class="container">
        {#each blogs as b}
            <article>
                <h1><a use:preload href="/blog/{b.ID}">{b.Title}</a></h1>
                <footer>
                    <small><a use:preload href="/user/{b.Author}">{b.Author}</a></small>
                    <small>{new Date(b.Timestamp * 1000).toLocaleString()}</small>
                </footer>
            </article>
        {/each}
    </div>
{/if}

<style>
    .container {
        display: flex;
        flex-direction: column-reverse;
        gap: 40px;
    }

    article {
        background: var(--bg-2);
        padding: 20px;
        width: 600px;
    }

    footer {
        display: flex;
        justify-content: space-between;
    }
</style>
