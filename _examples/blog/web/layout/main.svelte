<script>
    import { preload } from "golte";

    export let user;
</script>

<nav>
    <div class="left">
        <a use:preload href="/blog">Blog App</a>
    </div>
    <div class="right">
        {#if user}
            <a use:preload href="/new">New Blog</a>
            <a href="/logout" title="Logged in as {user}">Log Out</a>
        {:else}
            <a use:preload href="/login">Log In</a>
        {/if}
    </div>
</nav>

<main>
    <slot />
</main>

<style>
    nav {
        height: 80px;
        background: var(--bg-2);
        color: black;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0 200px;
    }

    a {
        color: inherit;
        text-decoration: none;
        padding: 20px 0;
    }

    a:hover {
        text-decoration: underline;
    }

    .left {
        font-size: 20px;
    }

    .right {
        font-size: 16px;
        display: flex;
        gap: 20px;
    }

    main {
        display: flex;
        justify-content: center;
        align-items: center;
        flex-direction: column;
        padding: 20px;
    }
</style>
