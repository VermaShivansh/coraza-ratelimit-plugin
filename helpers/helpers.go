package helpers

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"unicode"

	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
)

func NewWaf(conf string) coraza.WAF {
	if conf == "" {
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
	} else {
		waf, err := coraza.NewWAF(
			coraza.NewWAFConfig().
				// WithErrorCallback(logError).
				WithDirectives(conf),
		)
		if err != nil {
			log.Fatal(err)
		}
		return waf
	}
}

func NewHttpTestWafServer(conf string) *httptest.Server {
	waf := NewWaf(conf)

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

// checks if a string has
// minimum 1 letter
// mimimum 1 number
// minimum 16 letters
func CheckRatelimitDistributeKey(input string) error {
	if len(input) < 16 {
		return errors.New("distribute key must have minimum 16 alphanumeric characters")
	}

	hasNumber := false
	hasAlphabet := false

	for _, char := range input {
		if !hasNumber && unicode.IsNumber(char) {
			hasNumber = true
		} else if !hasAlphabet && unicode.IsLetter(char) {
			hasAlphabet = true
		}
		if hasNumber && hasAlphabet {
			break
		}
	}

	if !hasNumber {
		return errors.New("dstribute key must have atleast 1 number")
	}

	if !hasAlphabet {
		return errors.New("distribute key must have atleast 1 alphabet")
	}

	return nil
}
