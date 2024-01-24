<script>
    import { Link } from "../../../js/public/index.js"
</script>

<div id="layout2">
    <Link href="/route1" preload="mount">route1</Link>
    <Link href="/route2" preload="hover">route2</Link>
    <Link href="/route3" preload="tap"  >route3</Link>

    <slot/>
</div>
