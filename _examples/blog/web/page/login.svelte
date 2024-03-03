<script>
    import { url } from "golte/stores";

    let error;

    switch ($url.searchParams.get("err")) {
        case "1":
            error = "account already exists";
            break;
        case "2":
            error = "invalid credentials";
            break;
    }
</script>

<form method="post">
    <label>
        <span>Username</span>
        <input name="username" required/>
    </label>
    <label>
        <span>Password</span>
        <input name="password" type="password" required/>
    </label>
    <div>
        <button type="submit" formaction="/login">Log In</button>
        <button type="submit" formaction="/register">Sign Up</button>
        {#if error}<small>{error}</small>{/if}
    </div>
</form>

<style>
    form {
        padding: 20px;
        width: 500px;
        background: #8AB0BA;
        display: flex;
        flex-direction: column;
        align-items: center;
        border-radius: 10px;
        gap: 20px;
        line-height: 2;
    }

    form > * {
        width: 100%;
    }

    input {
        width: 100%;
        box-sizing: border-box;
    }

    small {
        color: red;
        float: right;
    }
</style>
