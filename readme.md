<h1>
  <img src="https://coraza.io/images/logo_shield_only.png" align="left" height="46px" alt=""/>&nbsp;
  <span>Coraza WAF - Ratelimit Plugin</span>
</h1> 

## Overview

**Ratelimit Plugin** for **Coraza Web Application Firewall**, aims to protect against brute Force attacks, DOS, DDOS attacks and prevent your servers from resource exhaustion. The plugin supports rate limitation on distributed servers as well.


## Installation

Add the plugin module to your application

```bash
  go get github.com/vermaShivansh/coraza-ratelimit-plugin
```

## Usage/Examples

#### Import the plugin

```go
import (
  "fmt"
  "github.com/corazawaf/coraza/v3"
	_ "github.com/vermaShivansh/coraza-ratelimit-plugin/plugin" // registers the plugin
)
```

#### Start a basic Coraza WAF Server.
Refer [here](https://github.com/corazawaf/coraza#----coraza---web-application-firewall)
 to know more about starting Coraza WAF server.

```go
func main() {
 // First we initialize our waf and our seclang parser
 waf, err := coraza.NewWAF(coraza.NewWAFConfig().
  WithDirectivesFromFile("./default.conf"))
 // Now we parse our rules
 if err != nil {
  fmt.Println(err)
 }

 // Then we create a transaction and assign some variables
    tx := waf.NewTransaction()
 defer func() {
  tx.ProcessLogging()
  tx.Close()
 }()
 tx.ProcessConnection("127.0.0.1", 8080, "127.0.0.1", 12345)

 // Finally we process the request headers phase, which may return an interruption
 if it := tx.ProcessRequestHeaders(); it != nil {
  fmt.Printf("Transaction was interrupted with status %d\n", it.Status)
 }
}
```

Manipulate Seclang configuration inside **'./default.conf'**


### 1. Configuration for Single Zone Ratelimit implementation

```seclang
SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.host}&events=200&window=1, pass, status:200"
```
Above configuration allows ***200 requests(events=200)***, ***per second(window=1)***, ***per different host(zone[]=%{REQUEST_HEADERS.host})***. Once the requests are exhausted the requests will be **denied with status 429 by default**, See [reference]() for customizations.

### 2. Configuration for Multizone Zone Ratelimit implementation

```seclang
SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.host}&zone[]=%{REQUEST_HEADERS.authorization}&events=200&window=1, pass, status:200"
```
Above configuration allows 200 requests, per second, ***per different zone(zone[]=%{REQUEST_HEADERS.host}&zone[]=%{REQUEST_HEADERS.host})***. <br/>
Zones work in **OR** manner i.e if 200 requests have been received with **same  authorization header value** but from **2 different host**(100 from HOST A, 100 from HOST B) then ratelimit should fail according to our cap of 200 requests, **however HOST A and HOST B still have 100 requests remaining** therefore requests won't be rate limited. 

### 3. Configuration for Distributed Ratelimit implementation

```seclang
SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.authorization}&events=200&window=1&distribute_interval=5, pass, status:200"
```
You can enable distributed ratelimit accross different instances of your application. <br/>
**NOTE** <br/>
* You must have **same ratelimit configuration** throughout the different servers.
* Your application must have a **value set for environment variable `coraza_ratelimit_key`**. Example `os.Setenv("coraza_ratelimit_key", "my_unique_key")`. It is because instances with same unique key are synced together. In order to promote the uniqueness of your key, we have set following conditions.


## API Reference

#### Checkout all the options available for ratelimit configuration

```Secrule
  SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.authorization}&events=5&window=6&interval=10&action=deny&status=403&distribute_interval=5, pass, status:200"
```

| Parameter | Type     | Value | Description                |
| :-------- | :------- | :-----| :------------------- |
| `zone[]` | `string` | **Required** | This can be either a [macro]() for dynamic ratelimit application or a fixed string. |
|  `events` | `integer`  | **Required** | Number of requests allowed in specified window. 1 event= 1 request  |
| `window` | `integer` | **Required**| Window in seconds in which max requests are allowed. Value should be greater than 0. |
| `interval` | `integer` | **Optional** (default: 5) | Time interval in seconds after which memory cleaning is attempted. |
| `action` | `enum` | **Optional** (default: 'drop') | Action to execute when ratelimit has been exceeded. Action is one of **'deny','drop' or 'redirect'**.|
| `status` | `integer` | **Optional** (default: 429) | HTTP Response status to be sent along with action when ratelimit has been reached. |
| `distribute_interval` | `integer` | **Optional** | Following field enables distributed ratelimit and syncs among the instances every given interval. By default it is not set (hence off). It uses the environment value for the field `coraza_ratelimit_key`. |

Failing to follow any of the above reference will result in an error ( which are very easily understandable ) while parsing the SecRule.



## Demo

You can find an example implementation [here](https://github.com/VermaShivansh/coraza-ratelimit-plugin/tree/master/example_app).


## Recommendations and mistakes to avoid

* Understanding and knowing all the [macros](https://coraza.io/docs/seclang/syntax/#macro-expansion) will be highly helpful for setting up dynamic rate limit based upon request data (headers, request body, args and etc).
* If you want to completely restrict access to a route for matching rule, set `events`=0.
* The application stores the events in memory and uses `interval` value for memory cleaning. It is recommended to neither set the value too high (60 seconds) nor too low (5 second as it will be overkill). Specific value highly depends upon the kind of traffic you are dealing with. Also it is advised to keep `interval` value more than `window`, as if `window` is 10 seconds and `interval` is 5 seconds there will be nothing to clean from memory 2 times.  
* Before you enforce rate limiting, you can have a period of observation and look at the max RPS of your different customer profiles. You might then typically set the rate limits at some reasonable margin higher than that max, assuming that any meaningful spike beyond this is probably not intentional.
* When using distributed mode, it is possible that instance might receive 2-3% of max requests than intended. This is because we don't stop processing incoming requests while the instance is syncing with redis. The downside to this approach is your backend may receive a little more traffic than was strictly defined, however distributed ratelimit will be eventually consistent. In most cases, this is probably the right call to offer the best experience for your API overall.

## Under the hood

### Algorithm

### Distributed Ratelimit

### Benchmarks
