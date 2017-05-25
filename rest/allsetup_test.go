package rest

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

var lastModifiedDate = time.Now()

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

var tmux = http.NewServeMux()
var server = httptest.NewServer(tmux)

var users []User

var userList = []string{
	"Hernan", "Mariana", "Matilda", "Juan", "Pedro", "John", "Axel", "Mateo",
}

var rb = RequestBuilder{
	BaseURL: server.URL,
}

func TestMain(m *testing.M) {

	setup()
	code := m.Run()
	//	teardown()
	os.Exit(code)
}

func setup() {
	rand.Seed(time.Now().UnixNano())

	users = make([]User, len(userList))
	for i, n := range userList {
		users[i] = User{Id: i + 1, Name: n}
	}

	//users
	tmux.HandleFunc("/user", allUsers)
	tmux.HandleFunc("/xml/user", usersXML)
	tmux.HandleFunc("/cache/user", usersCache)
	tmux.HandleFunc("/cache/expires/user", usersCacheWithExpires)
	tmux.HandleFunc("/cache/etag/user", usersEtag)
	tmux.HandleFunc("/cache/lastmodified/user", usersLastModified)
	tmux.HandleFunc("/slow/cache/user", slowUsersCache)
	tmux.HandleFunc("/slow/user", slowUsers)

	//One user
	tmux.HandleFunc("/user/", oneUser)

	//Header
	tmux.HandleFunc("/header", withHeader)
}

func withHeader(writer http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodGet {
		if h := req.Header.Get("X-Test"); h == "test" {
			return
		}
	}

	writer.WriteHeader(http.StatusBadRequest)
	return
}

func slowUsersCache(writer http.ResponseWriter, req *http.Request) {
	time.Sleep(30 * time.Millisecond)
	usersCache(writer, req)
}

func slowUsers(writer http.ResponseWriter, req *http.Request) {
	time.Sleep(10 * time.Millisecond)
	allUsers(writer, req)
}

func usersCache(writer http.ResponseWriter, req *http.Request) {

	// Get
	if req.Method == http.MethodGet {

		c := rand.Intn(2) + 1
		b, _ := json.Marshal(users)

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(c))
		writer.Write(b)
	}
}

func usersCacheWithExpires(writer http.ResponseWriter, req *http.Request) {

	// Get
	if req.Method == http.MethodGet {

		c := rand.Intn(2) + 1
		b, _ := json.Marshal(users)

		expires := time.Now().Add(time.Duration(c) * time.Second)

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Expires", expires.Format(httpDateFormat))
		writer.Write(b)
	}
}

func usersEtag(writer http.ResponseWriter, req *http.Request) {

	// Get
	if req.Method == http.MethodGet {

		etag := req.Header.Get("If-None-Match")

		if etag == "1234" {
			writer.WriteHeader(http.StatusNotModified)
			return
		}

		b, _ := json.Marshal(users)

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("ETag", "1234")
		writer.Write(b)

	}
}

func usersLastModified(writer http.ResponseWriter, req *http.Request) {

	// Get
	if req.Method == http.MethodGet {

		ifModifiedSince, err := time.Parse(httpDateFormat, req.Header.Get("If-Modified-Since"))

		if err == nil && ifModifiedSince.Sub(lastModifiedDate) == 0 {
			writer.WriteHeader(http.StatusNotModified)
			return
		}

		b, _ := json.Marshal(users)

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Last-Modified", lastModifiedDate.Format(httpDateFormat))
		writer.Write(b)

	}
}

func usersXML(writer http.ResponseWriter, req *http.Request) {

	// Get
	if req.Method == http.MethodGet {

		b, _ := xml.Marshal(users)

		writer.Header().Set("Content-Type", "application/xml")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Write(b)
	}

	// Post
	if req.Method == http.MethodPost {

		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		u := new(User)
		if err = xml.Unmarshal(b, u); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		u.Id = 3
		ub, _ := json.Marshal(u)

		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(http.StatusCreated)
		writer.Write(ub)

		return
	}
}

func oneUser(writer http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodGet {
		b, _ := json.Marshal(users[0])

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Write(b)
		return
	}

	// Put
	if req.Method == http.MethodPut || req.Method == http.MethodPatch {
		b, _ := json.Marshal(users[0])

		writer.Header().Set("Content-Type", "application/json")
		writer.Write(b)
		return
	}

	// Delete
	if req.Method == http.MethodDelete {
		return
	}
}

func allUsers(writer http.ResponseWriter, req *http.Request) {

	// Head
	if req.Method == http.MethodHead {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-cache")
		return
	}

	// Get
	if req.Method == http.MethodGet {

		b, _ := json.Marshal(users)

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Write(b)
		return
	}

	// Post
	if req.Method == http.MethodPost {

		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		u := new(User)
		if err = json.Unmarshal(b, u); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		u.Id = 3
		ub, _ := json.Marshal(u)

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		writer.Write(ub)

		return
	}

	// Options
	if req.Method == http.MethodOptions {

		b := []byte(`User resource
		id: Id of the user
		name: Name of the user`)

		writer.Header().Set("Content-Type", "text/plain")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Write(b)
		return
	}

}
