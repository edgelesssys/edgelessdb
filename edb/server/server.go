package server

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/edgelesssys/edb/edb/core"
)

// CreateServeMux creates a mux that serves the edb API.
func CreateServeMux(core *core.Core) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/manifest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		jsonManifest, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		recoveryKey, err := core.Initialize(jsonManifest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if recoveryKey != nil {
			io.WriteString(w, base64.StdEncoding.EncodeToString(recoveryKey))
		}
	})

	mux.HandleFunc("/signature", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, hex.EncodeToString(core.GetManifestSignature()))
	})

	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		report := core.GetReport()
		if len(report) == 0 {
			http.Error(w, "failed to get quote", http.StatusInternalServerError)
			return
		}
		w.Write(report)
	})

	return mux
}

// RunServer runs a HTTP server serving mux.
func RunServer(mux *http.ServeMux, address string, tlsConfig *tls.Config) {
	server := http.Server{
		Addr:      address,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	fmt.Println(server.ListenAndServeTLS("", ""))
}
