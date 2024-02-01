<script>
    import { preload } from "../../../js/public/index.js"
</script>

<div id="layout2">
    <a href="/route1" use:preload={"mount"}>route1</a>
    <a href="/route2" use:preload={"hover"}>route2</a>
    <a href="/route3" use:preload={"tap"}>route3</a>

    <slot/>
</div>
