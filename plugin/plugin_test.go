//go:generate tinygo build -o ./testdata/hello-world.wasm -target=wasi ./testdata/hello-world/main.go
package plugin_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/corazawaf/coraza/v3"
)

func TestRatelimit(t *testing.T) {
	waf, err := coraza.NewWAF(
		coraza.NewWAFConfig().
			WithDirectivesFromFile("../default.conf"),
	)
	require.NoError(t, err)

	tx := waf.NewTransaction()
	tx.ProcessRequestHeaders()
	tx.ProcessResponseBody()
	tx.ProcessResponseHeaders(200, "HTTP/1.1")
	tx.ProcessLogging()

	var tests = []struct {
		url                string
		expectedStatusCode int
	}{
		{"http://localhost:8090?id=2", 200},
		{"http://localhost:8090?id=2", 200},
		{"http://localhost:8090?id=2", 200},
		{"http://localhost:8090?id=2", 200},
		{"http://localhost:8090?id=2", 429},
		{"http://localhost:8090?id=2", 429},
		{"http://localhost:8090?id=2", 429},
		{"http://localhost:8090?id=2", 429},
		{"http://localhost:8090?id=2", 200},
		{"http://localhost:8090?id=2", 200},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test %v", i), func(t *testing.T) {
			res, err := http.Get(test.url)
			if err != nil {
				fmt.Printf("Error: %s", err)
				t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, err.Error())
			}

			if res.StatusCode != test.expectedStatusCode {
				t.Errorf("Error in %v, expected: %v, got: %v", test.url, test.expectedStatusCode, res.StatusCode)
			}
		})
		time.Sleep(125 * time.Millisecond)
	}

	tx.Close()
}
