<script>
    export let username;
    export let realname;
    export let occupation;
    export let age;
    export let email;
    export let site;
    export let searching;
</script>

<div class="name">
    <h2>{realname}</h2>
    <small>{occupation}</small>
</div>

<table>
    <tbody>
        <tr>
            <td>Username</td>
            <td>{username}</td>
        </tr>
        <tr>
            <td>Age</td>
            <td>{age}</td>
        </tr>
        <tr>
            <td>Email</td>
            <td>{email}</td>
        </tr>
        <tr>
            <td>Site</td>
            <td><a href={site}>{site}</a></td>
        </tr>
        <tr>
            <td>Looking for job?</td>
            <td>{searching ? "Yes" : "No"}</td>
        </tr>
    </tbody>
</table>

<style>
    h2 {
        margin-bottom: 0;
    }

    .name {
        padding-bottom: 20px;
    }

    tr td:first-of-type {
        font-weight: 700;
        padding-right: 20px;
    }
</style>
