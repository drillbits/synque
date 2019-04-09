// Copyright 2019 drillbits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package synque

import (
	"encoding/json"
	"net/http"
)

func NewServer(d *Dispatcher, addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, synque!"))
	})

	mux.HandleFunc("/queue", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.MarshalIndent(d.waiting, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	mux.HandleFunc("/queue/enqueue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var t Task
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		d.Enqueue(&t)

		b, err := json.MarshalIndent(d.waiting, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	mux.HandleFunc("/queue/dequeue", func(w http.ResponseWriter, r *http.Request) {
		// TODO: d.waiting.Delete(id)
		d.waiting.s.Range(func(k, v interface{}) bool {
			d.waiting.s.Delete(k)
			return true
		})

		b, err := json.MarshalIndent(d.waiting, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
