//go:generate tinygo build -o ./testdata/hello-world.wasm -target=wasi ./testdata/hello-world/main.go
package plugin

import (
	"testing"

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
	tx.Close()
}
