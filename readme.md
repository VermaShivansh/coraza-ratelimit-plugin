<h1>Coraza ratelimit plugin</h1>
<br/><br/>
SecRule ARGS:id "@eq 1" "id:2, ratelimit:10s, pass, status:200"
<br/>
Allows 10 requests per second for matching SecRule. If limit reaches request is denied with status code 429.
<br/>
There are lot more customizations to come. Its just a prototype
