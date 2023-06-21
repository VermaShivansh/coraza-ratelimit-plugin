package helpers

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
)

func NewWaf() coraza.WAF {
	directivesFile := "../default.conf"
	if s := os.Getenv("DIRECTIVES_FILE"); s != "" {
		directivesFile = s
	}

	waf, err := coraza.NewWAF(
		coraza.NewWAFConfig().
			// WithErrorCallback(logError).
			WithDirectivesFromFile(directivesFile),
	)
	if err != nil {
		log.Fatal(err)
	}
	return waf
}

func NewHttpTestWafServer() *httptest.Server {
	waf := NewWaf()

	// create server using httptest
	svr := httptest.NewServer(txhttp.WrapHandler(waf, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		resBody := "Transaction not disrupted."

		// The server generates the response
		_, err := w.Write([]byte(resBody))
		if err != nil {
			log.Fatal("Error in writing response body to header")
		}
	})))

	return svr
}
