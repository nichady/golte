<script>
    export let varNum, varStr, varBool;
</script>

<div id="layout1">
    <slot/>
</div>

<div id="layout1-varNum" val={varNum}/>
<div id="layout1-varStr" val={varStr}/>
<div id="layout1-varBool" val={varBool}/>
