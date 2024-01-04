<script>
    export let varNum, varStr, varBool;
</script>

<div id="layout1">
    <div id="varNum" val={varNum}/>
    <div id="varStr" val={varStr}/>
    <div id="varBool" val={varBool}/>
    
    <slot/>
</div>
