//go:generate tinygo build -o ./testdata/hello-world.wasm -target=wasi ./testdata/hello-world/main.go
package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vermaShivansh/coraza-ratelimit-plugin/helpers"
)

// tests ratelimit configuration
func TestConfigurationParser(t *testing.T) {

	// This is a complete Secrule
	// SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone=%{REQUEST_HEADERS.host}&events=200&window=1&interval=1&action=drop&status=429, pass, status:200"
	// we will only be testing parseConfig method using the value sent in ratelimit action
	testCases := ConfigTestCases

	ratelimit := &Ratelimit{}

	for _, testCase := range testCases {
		if testCase.Expected {
			assert.Equal(t, nil, ratelimit.parseConfig(testCase.Config))
		} else {
			assert.NotEqual(t, nil, ratelimit.parseConfig(testCase.Config))
		}
	}

	fmt.Println("Ratelimit configuration check passed")
}

// sends 200 requests every second to server. With Config maxEvents=200, sweepInterval=2, window=1
// all requests should give 200
func TestLogicOfRateLimit(t *testing.T) {
	wg := &sync.WaitGroup{}

	// get an instance of http test server with waf
	conf := `SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.host}&events=200&window=1&interval=2&action=deny&status=403, pass, status:200"`

	svr := helpers.NewHttpTestWafServer(conf)
	defer svr.Close()

	requestURL := fmt.Sprintf("%v?id=1", svr.URL)

	ticker := time.NewTicker(time.Second * 1)

	i := 0

	for {
		<-ticker.C

		for i := 0; i < 200; i++ {

			wg.Add(1)
			go func(wg *sync.WaitGroup, i int) {
				defer wg.Done()
				for j := 0; j < 1; j++ {
					resp, err := svr.Client().Get(requestURL)
					if err != nil {
						t.Errorf("Error in %v, expected: %v, got: %v", i, 200, err.Error())
					}
					assert.Equal(t, 200, resp.StatusCode)
				}
			}(wg, i)
		}

		i++

		if i == 5 {
			ticker.Stop()
			break
		}

	}

	wg.Wait()
}

// 1000 requests in a second should be executed successfully.
// currently it is taking 625ms to execute 1000req/sec
func TestStressOfRateLimit(t *testing.T) {
	wg := &sync.WaitGroup{}

	// get an instance of http test server with waf
	conf := `SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.host}&events=300&window=1&interval=2&action=deny&status=429, pass, status:200"`

	svr := helpers.NewHttpTestWafServer(conf)
	defer svr.Close()

	requestURL := fmt.Sprintf("%v?id=1", svr.URL)

	currentTime := time.Now().UnixMilli()

	for i := 0; i < 1000; i++ {

		wg.Add(1)
		go func(wg *sync.WaitGroup, i int) {
			defer wg.Done()
			for j := 0; j < 1; j++ {
				_, err := svr.Client().Get(requestURL)
				if err != nil {
					t.Errorf("Error in %v, expected: %v, got: %v", i, 200, err.Error())
				}
				// assert.Equal(t, 200, resp.StatusCode)
			}
		}(wg, i)
	}

	wg.Wait()

	timeTaken := time.Now().UnixMilli() - currentTime

	assert.LessOrEqual(t, timeTaken, int64(1000))

	fmt.Printf("Time taken to execute 1000 goroutines sending 1request at server is: %v\n", timeTaken)
}

// host zone remains same in all request but zone dependent on macro queryString changes
// total 4 different types of queryStrings are used ?id=1&category=0/1/2/3
// overall 48 requests are executed in 1 second and 11 events are allowed in 1 second window
// after first 11 requests-> request will fail according to zone HOST but will be allowed as it won't exceed 11 requests per zone according to queryString zone (each zone would still have approx 10 reqs remaining)
// last 4 reqs will fail for each different zone based on queryString as only 11reqs are allowed but these last reqs would be the 12th request.
func TestMultiZone(t *testing.T) {
	wg := &sync.WaitGroup{}

	results := []string{}
	mut := &sync.Mutex{}
	initialTime := time.Now()

	// get an instance of http test server with waf
	conf := `SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=%{REQUEST_HEADERS.host}&zone[]=%{QUERY_STRING}&events=11&window=1&interval=2&action=deny&status=401, pass, status:200"`

	svr := helpers.NewHttpTestWafServer(conf)
	defer svr.Close()

	failedReqs := 0

	for i := 0; i < 48; i++ {
		wg.Add(1)
		requestURL := fmt.Sprintf("%v?id=1&category=%v", svr.URL, i%4)
		go func(wg *sync.WaitGroup, mut *sync.Mutex, i int) {
			defer wg.Done()
			resp, err := svr.Client().Get(requestURL)
			if err != nil {
				fmt.Printf("Error: %s", err)
				// t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
			}
			if resp.StatusCode == 200 {
				mut.Lock()
				results = append(results, fmt.Sprintf("PASS: i=%v, time=%v", i, time.Since(initialTime).Milliseconds()))
				mut.Unlock()
			} else {
				mut.Lock()
				failedReqs++
				results = append(results, fmt.Sprintf("FAIL: i=%v, time=%v", i, time.Since(initialTime).Milliseconds()))
				mut.Unlock()
			}
		}(wg, mut, i)
	}
	wg.Wait()

	prettyPrint(results)

	assert.Equal(t, 4, failedReqs, "only 4 reqs should fail according to the logic")
}

func TestDistributedSystemsSupport(t *testing.T) {

	// get an instance of http test server with waf
	conf := `SecRule ARGS:id "@eq 1" "id:1, ratelimit:zone[]=fixed&events=5&window=6&interval=10&action=deny&status=403, pass, status:200"`

	//client server which will request to running WAF instances
	clientSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// 3 WAF instances
	svr1 := helpers.NewHttpTestWafServer(conf)
	defer svr1.Close()

	svr2 := helpers.NewHttpTestWafServer(conf)
	defer svr2.Close()

	svr3 := helpers.NewHttpTestWafServer(conf)
	defer svr3.Close()

	fmt.Println(svr1.URL, svr2.URL, svr3.URL)

	// 1 req at a time to all 3 servers
	for {
		if _, err := clientSvr.Client().Get(fmt.Sprintf("%v?id=1", svr1.URL)); err != nil {
			log.Println(err)
		}
		if _, err := clientSvr.Client().Get(fmt.Sprintf("%v?id=1", svr2.URL)); err != nil {
			log.Println(err)
		}
		if _, err := clientSvr.Client().Get(fmt.Sprintf("%v?id=1", svr3.URL)); err != nil {
			log.Println(err)
		}
		time.Sleep(1 * time.Second)
	}
}

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}
