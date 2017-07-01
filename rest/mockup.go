package rest

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const MOCK_NOT_FOUND_ERROR string = "MockUp nil!"

var mockUpEnv bool
var mockMap = make(map[string]*Mock)
var mockDbMutex sync.RWMutex

var mockServer *httptest.Server
var mux *http.ServeMux

var mockServerURL *url.URL

// Mock serves the purpose of creating Mockups.
// All requests will be sent to the mockup server if mockup is activated.
// To activate the mockup *environment* you have two ways: using the flag -mock
//	go test -mock
//
// Or by programmatically starting the mockup server
// 	StartMockupServer()
type Mock struct {

	// Request URL
	URL string

	// Request HTTP Method (GET, POST, PUT, PATCH, HEAD, DELETE, OPTIONS)
	// As a good practice use the constants in http package (http.MethodGet, etc.)
	HTTPMethod string

	// Request array Headers
	ReqHeaders http.Header

	// Request Body, used with POST, PUT & PATCH
	ReqBody string

	// Response HTTP Code
	RespHTTPCode int

	// Response Array Headers
	RespHeaders http.Header

	// Response Body
	RespBody string
}

// StartMockupServer sets the enviroment to send all client requests
// to the mockup server.
func StartMockupServer() {

	mockUpEnv = true

	if mockServer == nil {
		startMockupServ()
	}
}

// StopMockupServer stop sending requests to the mockup server
func StopMockupServer() {

	mockUpEnv = false
	mockServer.Close()

	mockServer = nil
	mockServerURL = nil
	mux = nil
}

func startMockupServ() {

	if mockUpEnv {
		mux = http.NewServeMux()
		mockServer = httptest.NewServer(mux)
		mux.HandleFunc("/", mockupHandler)
		mockDbMutex = *new(sync.RWMutex)

		var err error
		if mockServerURL, err = url.Parse(mockServer.URL); err != nil {
			panic(err)
		}

	}
}

func init() {
	flag.BoolVar(&mockUpEnv, "mock", false,
		"Use 'mock' flag to tell package rest that you would like to use mockups.")

	flag.Parse()
	startMockupServ()
}

// AddMockups ...
func AddMockups(mocks ...*Mock) error {
	for _, m := range mocks {
		normalizedUrl, err := getNormalizedUrl(m.URL)
		if err != nil {
			return errors.New(fmt.Sprintf("Error parsing mock with url=%s. Cause: %s", m.URL, err.Error()))
		}
		mockDbMutex.Lock()
		mockMap[m.HTTPMethod+" "+normalizedUrl] = m
		mockDbMutex.Unlock()
	}
	return nil
}

//check if a string url is valid and also sort query params in order to make the url easy to compare
func getNormalizedUrl(urlStr string) (string, error) {
	urlObj, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	result := urlStr

	//sorting query param strings
	if len(urlObj.RawQuery) > 0 {
		result = strings.Replace(urlStr, urlObj.RawQuery, "", 1)

		mk := make([]string, len(urlObj.Query()))
		i := 0
		for k, _ := range urlObj.Query() {
			mk[i] = k
			i++
		}
		sort.Strings(mk)
		for i := 0; i < len(mk); i++ {
			if i+1 < len(mk) {
				result = fmt.Sprintf("%s%s=%s&", result, mk[i], urlObj.Query().Get(mk[i]))
			} else {
				result = fmt.Sprintf("%s%s=%s", result, mk[i], urlObj.Query().Get(mk[i]))
			}
		}
	}
	return result, nil
}

// FlushMockups ...
func FlushMockups() {
	mockDbMutex.Lock()
	mockMap = make(map[string]*Mock)
	mockDbMutex.Unlock()
}

func mockupHandler(writer http.ResponseWriter, req *http.Request) {

	normalizedUrl, err := getNormalizedUrl(req.Header.Get("X-Original-URL"))

	if err == nil {
		mockDbMutex.RLock()
		m := mockMap[req.Method+" "+normalizedUrl]
		mockDbMutex.RUnlock()
		if m != nil {
			// Add headers
			for k, v := range m.RespHeaders {
				for _, vv := range v {
					writer.Header().Add(k, vv)
				}
			}

			writer.WriteHeader(m.RespHTTPCode)
			writer.Write([]byte(m.RespBody))
			return
		}
	}

	writer.WriteHeader(http.StatusBadRequest)
	writer.Write([]byte(MOCK_NOT_FOUND_ERROR))
}
