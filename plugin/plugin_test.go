//go:generate tinygo build -o ./testdata/hello-world.wasm -target=wasi ./testdata/hello-world/main.go
package plugin

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/vermaShivansh/coraza-ratelimit-plugin/helpers"
)

// sends 200 requests every second to server. With Config maxEvents=200, sweepInterval=2, window=1 // this should work perfectly fine
func TestLogicOfRateLimit(t *testing.T) {
	wg := &sync.WaitGroup{}

	results := []string{}
	mut := &sync.Mutex{}
	initialTime := time.Now()

	// get an instance of http test server with waf
	svr := helpers.NewHttpTestWafServer()
	defer svr.Close()

	requestURL := fmt.Sprintf("%v?id=1", svr.URL)
	log.Println("requestURL", requestURL)

	for i := 0; i < 1000; i++ {
		if i%200 == 0 {
			time.Sleep(time.Second * 1)
		}
		wg.Add(1)
		go func(wg *sync.WaitGroup, mut *sync.Mutex, i int) {
			defer wg.Done()
			for j := 0; j < 1; j++ {
				resp, err := svr.Client().Get(requestURL)
				if err != nil {
					fmt.Printf("Error: %s", err)
					// t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
				}
				if resp.StatusCode == 200 {
					mut.Lock()
					results = append(results, fmt.Sprintf("PASS: i=%v, j=%v, time=%v", i, j, time.Since(initialTime).Milliseconds()))
					mut.Unlock()
				} else {
					mut.Lock()
					results = append(results, fmt.Sprintf("FAIL: i=%v, j=%v, time=%v", i, j, time.Since(initialTime).Milliseconds()))
					mut.Unlock()
				}
			}
		}(wg, mut, i)
	}
	wg.Wait()
	prettyPrint(results)
}

// 1000 requests in a second
func TestStressOfRateLimit(t *testing.T) {
	wg := &sync.WaitGroup{}

	results := []string{}
	mut := &sync.Mutex{}
	initialTime := time.Now()

	// get an instance of http test server with waf
	svr := helpers.NewHttpTestWafServer()
	defer svr.Close()

	requestURL := fmt.Sprintf("%v?id=1", svr.URL)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, mut *sync.Mutex, i int) {
			defer wg.Done()
			for j := 0; j < 1; j++ {
				resp, err := svr.Client().Get(requestURL)
				if err != nil {
					fmt.Printf("Error: %s", err)
					// t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
				}
				if resp.StatusCode == 200 {
					mut.Lock()
					results = append(results, fmt.Sprintf("PASS: i=%v, j=%v, time=%v", i, j, time.Since(initialTime).Milliseconds()))
					mut.Unlock()
				} else {
					mut.Lock()
					results = append(results, fmt.Sprintf("FAIL: i=%v, j=%v, time=%v", i, j, time.Since(initialTime).Milliseconds()))
					mut.Unlock()
				}
			}
		}(wg, mut, i)
	}
	wg.Wait()
	prettyPrint(results)
}
