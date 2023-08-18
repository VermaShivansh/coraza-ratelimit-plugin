package main

import (
	"log"
	"net/http"
	"os"

	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"

	_ "github.com/vermaShivansh/coraza-ratelimit-plugin/plugin" // registers the plugin
)

func main() {
	// it is recommended to use dotenv or some other package to store envs
	os.Setenv("coraza_ratelimit_key", "this_key_is_unique_and_same_for_all_my_instances")
	os.Setenv("PORT", ":8080")

	// First we initialize waf
	directivesFile := "./default.conf"
	if s := os.Getenv("DIRECTIVES_FILE"); s != "" {
		directivesFile = s
	}
	waf, err := coraza.NewWAF(
		coraza.NewWAFConfig().
			WithDirectivesFromFile(directivesFile),
	)
	if err != nil {
		log.Fatal(err)
	}

	// add a middleware to handle the incoming requests
	if err := http.ListenAndServe(os.Getenv("PORT"), txhttp.WrapHandler(waf, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		resBody := "Transaction not disrupted."

		// The server generates the response
		_, err := w.Write([]byte(resBody))
		if err != nil {
			log.Fatal("Error in writing response body to header")
		}
	}))); err != nil {
		log.Fatal(err)
	}

}
