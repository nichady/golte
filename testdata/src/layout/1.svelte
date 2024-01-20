<script>
    export let varNum, varStr, varBool;
</script>

<div id="layout1">
    <div id="props" {varNum} {varStr} {varBool} />
    <slot/>
</div>
