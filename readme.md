<h1>Coraza ratelimit plugin</h1>
<br/>
<hr/>
<br/>
<h3><bold>Examples:- </bold>  </h3> 
<br/>
<ul>
<li>
<code>SecRule ARGS:id "@eq 1" "id:2, ratelimit:10s, pass, status:200"</code><br/>
Allows 10 requests per second for matching SecRule (Query String parameter value of id=1). If limit reaches request is denied with status code 429.
</li>
<br/>
<li>
<code>SecRule REMOTE_ADDR "34.123.14" "id:2, ratelimit:100m, pass, status:200"</code><br/>
Allows 100 requests per minute for requests from REMOTE_ADDR=34.123.14. If limit reaches request is denied with status code 429.
</li>
</ul> 
<br/>
<bold>There are lot more customizations to come. Its just a prototype.</bold>
