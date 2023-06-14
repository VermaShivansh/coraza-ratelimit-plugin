//go:generate tinygo build -o ./testdata/hello-world.wasm -target=wasi ./testdata/hello-world/main.go
package plugin_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// func TestRatelimit(t *testing.T) {
// 	// waf, err := coraza.NewWAF(
// 	// 	coraza.NewWAFConfig().
// 	// 		WithDirectivesFromFile("../default.conf"),
// 	// )
// 	// require.NoError(t, err)

// 	// tx := waf.NewTransaction()
// 	// tx.ProcessRequestHeaders()
// 	// tx.ProcessResponseBody()
// 	// tx.ProcessResponseHeaders(200, "HTTP/1.1")
// 	// tx.ProcessLogging()
// 	// defer tx.Close()

// 	var tests = []struct {
// 		url                string
// 		expectedStatusCode int
// 	}{
// 		{"http://localhost:8090?id=2", 200},
// 		{"http://localhost:8090?id=2", 200},
// 		{"http://localhost:8090?id=2", 200},
// 		{"http://localhost:8090?id=2", 200},
// 		{"http://localhost:8090?id=2", 429},
// 		{"http://localhost:8090?id=2", 429},
// 		{"http://localhost:8090?id=2", 429},
// 		{"http://localhost:8090?id=2", 429},
// 		{"http://localhost:8090?id=2", 200},
// 		{"http://localhost:8090?id=2", 200},
// 	}

// 	for i, test := range tests {
// 		t.Run(fmt.Sprintf("Test %v", i), func(t *testing.T) {
// 			res, err := http.Get(test.url)
// 			if err != nil {
// 				fmt.Printf("Error: %s", err)
// 				t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
// 			}

// 			if res.StatusCode != test.expectedStatusCode {
// 				t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, res.StatusCode)
// 			}
// 		})
// 		time.Sleep(125 * time.Millisecond)
// 	}

// }

// sends 200 requests every second to server. With Config maxEvents=200, sweepInterval=2, window=1 // this should work perfectly fine
func TestLogicOfRateLimit(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1000)

	results := []string{}
	mut := &sync.Mutex{}
	initialTime := time.Now()
	for i := 0; i < 1000; i++ {
		if i%200 == 0 {
			time.Sleep(time.Second * 1)
		}
		go func(wg *sync.WaitGroup, mut *sync.Mutex, i int) {
			defer wg.Done()
			for j := 0; j < 1; j++ {
				resp, err := http.Get("http://localhost:8090?id=1")
				if err != nil {
					fmt.Printf("Error: %s", err)
					// t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
				}
				if resp.StatusCode == 200 {
					mut.Lock()
					results = append(results, fmt.Sprintf("PASS: i=%v, j=%v, time=%v", i, j, time.Since(initialTime).Milliseconds()))
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
	wg.Add(1000)

	results := []string{}
	mut := &sync.Mutex{}
	initialTime := time.Now()
	for i := 0; i < 1000; i++ {
		go func(wg *sync.WaitGroup, mut *sync.Mutex, i int) {
			defer wg.Done()
			for j := 0; j < 1; j++ {
				resp, err := http.Get("http://localhost:8090?id=1")
				if err != nil {
					fmt.Printf("Error: %s", err)
					// t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
				}
				if resp.StatusCode == 200 {
					mut.Lock()
					results = append(results, fmt.Sprintf("PASS: i=%v, j=%v, time=%v", i, j, time.Since(initialTime).Milliseconds()))
					mut.Unlock()
				}
			}
		}(wg, mut, i)
	}
	wg.Wait()
	prettyPrint(results)
}

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}
