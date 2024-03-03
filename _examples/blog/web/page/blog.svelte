<script>
    import { preload } from "golte";

    export let blog;
</script>

<h1>{blog.Title}</h1>
<small><a use:preload href="/user/{blog.Author}">{blog.Author}</a></small>
<p>{blog.Body}</p>

<style>
    p {
        white-space: pre-wrap;
    }
</style>
