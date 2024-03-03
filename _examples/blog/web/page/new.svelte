<h1>New Blog Post</h1>

<form method="post" action="/blog">
    <input placeholder="Title" name="title" required/>
    <textarea name="body" required/>
    <span>
        <button type="submit">Publish</button>
    </span>
</form>

<style>
    form {
        width: 1000px;
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    textarea {
        resize: none;
        overflow-y: scroll;
        height: 400px;
    }
</style>
