// Copyright 2012-2014 Diwaker Gupta
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

package gaestebin

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/user"
)

func init() {
	http.Handle("/paste/", new(PasteHandler))
}

type Paste struct {
	// Generated
	Id        string    `datastore:"id"`
	Timestamp time.Time `datastore:"timestamp"`
	// Required
	Content string `datastore:"content,noindex"`
	Email   string `datastore:"email"`
	// Optional/Best-effort
	Title    string `datastore:"title"`
	Language string `datastore:"language"`
	// Used for deletes, not persisted
	IsOwner bool `datastore:"-"`
}

// Handler for Paste API
type PasteHandler struct {
}

func (handler PasteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		c.Infof("%v Login required", appengine.RequestID(c))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Regexp used to extract pasteId from URI path
	idRegexp := regexp.MustCompile(`/paste/v1/([^/]+)`)
	switch r.Method {
	// GET /paste/<id>
	case "GET":
		// First extract the pasteId
		match := idRegexp.FindStringSubmatch(r.URL.Path)
		if match == nil {
			http.Error(w, "No pasteId found, bad URL", http.StatusBadRequest)
			return
		}
		pasteId := match[1]
		var paste Paste
		// First, lookup in memcache
		_, err := memcache.JSON.Get(c, pasteId, &paste)
		// If there's a miss, check in datastore
		if err == memcache.ErrCacheMiss {
			// First look up directly by pasteId for new pastes
			key := datastore.NewKey(c, "Paste", pasteId, 0, nil)
			err := datastore.Get(c, key, &paste)
			// If not found, try again with a query for v1 pastes
			if err != nil {
				q := datastore.NewQuery("Paste").Filter("id =", pasteId)
				var pastes []Paste
				_, err := q.GetAll(c, &pastes)
				if err != nil || len(pastes) == 0 {
					http.Error(w, "Paste not found", http.StatusNotFound)
					return
				}
				paste = pastes[0]
			}
			item := &memcache.Item{Key: pasteId, Object: paste}
			memcache.JSON.Set(c, item)
			c.Infof("Adding %v to memcache", pasteId)
		} else {
			c.Infof("Found %v in memcache", pasteId)
		}
		paste.IsOwner = (paste.Email == u.Email)

		// Send paste back as JSON
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(&paste); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	// POST /paste/ returns <id>
	case "POST":
		// Decode request
		decoder := json.NewDecoder(r.Body)
		var paste Paste
		if err := decoder.Decode(&paste); err != nil {
			c.Infof("Error decoding request %v", r)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Assign id, timestamp, email
		paste.Id = GenerateRandomString(8)
		paste.Timestamp = time.Now()
		paste.Email = u.Email
		paste.IsOwner = (paste.Email == u.Email)

		// Create a key using pasteId and save to datastore
		key := datastore.NewKey(c, "Paste", paste.Id, 0, nil)
		_, err := datastore.Put(c, key, &paste)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Additionally insert in memcache
		item := &memcache.Item{Key: paste.Id, Object: paste}
		memcache.JSON.Set(c, item)

		// Send back the complete paste
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(&paste); err != nil {
			c.Infof(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	// DELETE /paste/<id>
	case "DELETE":
		match := idRegexp.FindStringSubmatch(r.URL.Path)
		if match == nil {
			http.Error(w, "No pasteId found, bad URL", http.StatusBadRequest)
			return
		}
		pasteId := match[1]
		key := datastore.NewKey(c, "Paste", pasteId, 0, nil)

		// Look up the paste to match user
		var paste Paste
		if err := datastore.Get(c, key, &paste); err != nil {
			c.Infof(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if paste.Email != u.Email {
			c.Infof("Bad owner")
			http.Error(w, "Bad Owner", http.StatusForbidden)
			return
		}

		if err := datastore.Delete(c, key); err != nil {
			c.Infof(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		memcache.Delete(c, pasteId)
	}
}

func GenerateRandomString(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().Unix())
	str := make([]string, length)
	for i := 0; i < length; i++ {
		index := rand.Intn(len(chars))
		str[i] = chars[index : index+1]
	}
	return strings.Join(str, "")
}
