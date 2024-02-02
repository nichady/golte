<script>
    import { preload } from "golte";
    import { url } from "golte/stores";
</script>

<header>
    <nav>
        <ul>
            <li aria-current={$url.pathname === "/" ? "page" : undefined}>
                <a href="/" use:preload>Home</a>
            </li>
            <li aria-current={$url.pathname === "/about" ? "page" : undefined}>
                <a href="/about" use:preload>About</a>
            </li>
            <li aria-current={$url.pathname === "/contact" ? "page" : undefined}>
                <a href="/contact" use:preload>Contact</a>
            </li>
            <li aria-current={$url.pathname === "/user/profile" ? "page" : undefined}>
                <a href="/user/profile" use:preload>Profile</a>
            </li>
            <li aria-current={$url.pathname === "/user/login" ? "page" : undefined}>
                <a href="/user/login" use:preload>Login</a>
            </li>
        </ul>
    </nav>
</header>

<main>
    <slot />
</main>

<style>
    main {
        padding: 20px;
    }

    ul {
        margin: 0;
        height: 60px;

        list-style: none;
		background: #ccddf8;;
        
        display: flex;
        gap: 10px;
        justify-content: center;
        align-items: center;
    }

    li {
        height: 40px;
        padding: 0 10px;
    }

    li[aria-current="page"] {
        border-bottom: solid 2px orangered;
	}
    
    a {
        height: 100%;
        display: flex;
        align-items: center;
        color: inherit;
        
        font-family: "Gill Sans", sans-serif;;
        font-weight: 700;
		font-size: 0.9rem;
		text-decoration: none;
    }

    a:hover {
        color: orangered;
    }
</style>
