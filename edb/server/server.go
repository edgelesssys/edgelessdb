/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

package server

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/edgelesssys/edgelessdb/edb/core"
)

type generalResponse struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"` // only used when status = "error"
}

type certQuoteResp struct {
	Cert  string
	Quote []byte
}

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
		sig := core.GetManifestSignature()
		io.WriteString(w, hex.EncodeToString(sig))
	})

	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		cert, report, err := core.GetCertificateReport()
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, certQuoteResp{cert, report})
	})

	mux.HandleFunc("/recovery", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		key, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var statusMsg string
		if err := core.Recover(r.Context(), key); err != nil {
			statusMsg = fmt.Sprintf("Recovery failed: %v", err.Error())
		} else {
			statusMsg = "Recovery successful."
		}
		writeJSON(w, statusMsg)
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

func writeJSON(w http.ResponseWriter, v interface{}) {
	dataToReturn := generalResponse{Status: "success", Data: v}
	if err := json.NewEncoder(w).Encode(dataToReturn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSONError(w http.ResponseWriter, errorString string, httpErrorCode int) {
	marshalledJSON, err := json.Marshal(generalResponse{Status: "error", Message: errorString})
	// Only fall back to non-JSON error when we cannot even marshal the error (which is pretty bad)
	if err != nil {
		http.Error(w, errorString, httpErrorCode)
	}
	http.Error(w, string(marshalledJSON), httpErrorCode)
}
