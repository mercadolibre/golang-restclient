package rest

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var readVerbs = [3]string{http.MethodGet, http.MethodHead, http.MethodOptions}
var contentVerbs = [3]string{http.MethodPost, http.MethodPut, http.MethodPatch}
var defaultCheckRedirectFunc func(req *http.Request, via []*http.Request) error

var maxAge = regexp.MustCompile(`(?:max-age|s-maxage)=(\d+)`)

const httpDateFormat string = "Mon, 01 Jan 2006 15:04:05 GMT"

func (rb *RequestBuilder) doRequest(verb string, reqURL string, reqBody interface{}) (result *Response) {
	var cacheURL string
	var cacheResp *Response

	result = new(Response)
	reqURL = rb.BaseURL + reqURL

	//If Cache enable && operation is read: Cache GET
	if !rb.DisableCache && matchVerbs(verb, readVerbs) {
		if cacheResp = resourceCache.get(reqURL); cacheResp != nil {
			cacheResp.cacheHit.Store(true)
			if !cacheResp.revalidate {
				return cacheResp
			}
		}
	}

	func(verb string, reqURL string, reqBody interface{}) {

		//Marshal request to JSON or XML
		body, err := rb.marshalReqBody(reqBody)
		if err != nil {
			result.Err = err
			return
		}

		// Change URL to point to Mockup server
		reqURL, cacheURL, err = checkMockup(reqURL)
		if err != nil {
			result.Err = err
			return
		}

		//Get Client (client + transport)
		client := rb.getClient()

		request, err := http.NewRequest(verb, reqURL, bytes.NewBuffer(body))
		if err != nil {
			result.Err = err
			return
		}

		// Set extra parameters
		rb.setParams(request, cacheResp, cacheURL)

		// Make the request
		httpResp, err := client.Do(request)
		if err != nil {
			result.Err = err
			return
		}

		// Read response
		defer httpResp.Body.Close()
		respBody, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			result.Err = err
			return
		}

		// If we get a 304, return response from cache
		if httpResp.StatusCode == http.StatusNotModified {
			result = cacheResp
			return
		}

		result.Response = httpResp
		result.byteBody = respBody

		ttl := setTTL(result)
		lastModified := setLastModified(result)
		etag := setETag(result)

		if !ttl && (lastModified || etag) {
			result.revalidate = true
		}

		//If Cache enable: Cache SETNX
		if !rb.DisableCache && matchVerbs(verb, readVerbs) && (ttl || lastModified || etag) {
			resourceCache.setNX(cacheURL, result)
		}
		return
	}(verb, reqURL, reqBody)

	return

}

func checkMockup(reqURL string) (string, string, error) {

	cacheURL := reqURL

	if mockUpEnv {

		rURL, err := url.Parse(reqURL)
		if err != nil {
			return reqURL, cacheURL, err
		}

		rURL.Scheme = mockServerURL.Scheme
		rURL.Host = mockServerURL.Host

		return rURL.String(), cacheURL, nil
	}

	return reqURL, cacheURL, nil
}

func (rb *RequestBuilder) marshalReqBody(body interface{}) (b []byte, err error) {

	if body != nil {
		switch rb.ContentType {
		case JSON:
			b, err = json.Marshal(body)
		case XML:
			b, err = xml.Marshal(body)
		case BYTES:
			var ok bool
			b, ok = body.([]byte)
			if !ok {
				err = fmt.Errorf("bytes: body is %T(%v) not a byte slice", body, body)
			}
		}
	}

	return
}

func (rb *RequestBuilder) getClient() *http.Client {

	// This will be executed only once
	// per request builder
	rb.clientMtxOnce.Do(func() {

		dTransportMtxOnce.Do(func() {

			if defaultTransport == nil {
				defaultTransport = &http.Transport{
					MaxIdleConnsPerHost:   DefaultMaxIdleConnsPerHost,
					Proxy:                 http.ProxyFromEnvironment,
					DialContext:           (&net.Dialer{Timeout: rb.getConnectionTimeout()}).DialContext,
					ResponseHeaderTimeout: rb.getRequestTimeout(),
				}
			}

			defaultCheckRedirectFunc = http.Client{}.CheckRedirect
		})

		tr := defaultTransport

		if cp := rb.CustomPool; cp != nil {
			if cp.Transport == nil {
				tr = &http.Transport{
					MaxIdleConnsPerHost:   rb.CustomPool.MaxIdleConnsPerHost,
					DialContext:           (&net.Dialer{Timeout: rb.getConnectionTimeout()}).DialContext,
					ResponseHeaderTimeout: rb.getRequestTimeout(),
				}

				//Set Proxy
				if cp.Proxy != "" {
					if proxy, err := url.Parse(cp.Proxy); err == nil {
						tr.(*http.Transport).Proxy = http.ProxyURL(proxy)
					}
				}
				cp.Transport = tr

			} else {
				ctr, ok := cp.Transport.(*http.Transport)
				if ok {
					ctr.DialContext = (&net.Dialer{Timeout: rb.getConnectionTimeout()}).DialContext
					ctr.ResponseHeaderTimeout = rb.getRequestTimeout()
					tr = ctr
				} else {
					// If custom transport is not http.Transport, timeouts will not be overwritten.
					tr = cp.Transport
				}
			}
		}

		rb.Client = &http.Client{Transport: tr}

	})

	if !rb.FollowRedirect {
		rb.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return errors.New("Avoided redirect attempt")
		}
	} else {
		rb.Client.CheckRedirect = defaultCheckRedirectFunc
	}

	return rb.Client
}

func (rb *RequestBuilder) getRequestTimeout() time.Duration {

	switch {
	case rb.DisableTimeout:
		return 0
	case rb.Timeout > 0:
		return rb.Timeout
	default:
		return DefaultTimeout
	}
}

func (rb *RequestBuilder) getConnectionTimeout() time.Duration {

	switch {
	case rb.DisableTimeout:
		return 0
	case rb.ConnectTimeout > 0:
		return rb.ConnectTimeout
	default:
		return DefaultConnectTimeout
	}
}

func (rb *RequestBuilder) setParams(req *http.Request, cacheResp *Response, cacheURL string) {

	//Custom Headers
	if rb.Headers != nil {
		for key, values := range map[string][]string(rb.Headers) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	//Default headers
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")

	//If mockup
	if mockUpEnv {
		req.Header.Set("X-Original-URL", cacheURL)
	}

	// Basic Auth
	if rb.BasicAuth != nil {
		req.SetBasicAuth(rb.BasicAuth.UserName, rb.BasicAuth.Password)
	}

	// User Agent
	req.Header.Set("User-Agent", func() string {
		if rb.UserAgent != "" {
			return rb.UserAgent
		}
		return "github.com/go-loco/restful"
	}())

	//Encoding
	var cType string

	switch rb.ContentType {
	case JSON:
		cType = "json"
	case XML:
		cType = "xml"
	}

	if cType != "" {
		req.Header.Set("Accept", "application/"+cType)

		if matchVerbs(req.Method, contentVerbs) {
			req.Header.Set("Content-Type", "application/"+cType)
		}
	}

	if cacheResp != nil && cacheResp.revalidate {
		switch {
		case cacheResp.etag != "":
			req.Header.Set("If-None-Match", cacheResp.etag)
		case cacheResp.lastModified != nil:
			req.Header.Set("If-Modified-Since", cacheResp.lastModified.Format(httpDateFormat))
		}
	}

}

func matchVerbs(s string, sarray [3]string) bool {
	for i := 0; i < len(sarray); i++ {
		if sarray[i] == s {
			return true
		}
	}

	return false
}

func setTTL(resp *Response) (set bool) {

	now := time.Now()

	//Cache-Control Header
	cacheControl := maxAge.FindStringSubmatch(resp.Header.Get("Cache-Control"))

	if len(cacheControl) > 1 {

		ttl, err := strconv.Atoi(cacheControl[1])
		if err != nil {
			return
		}

		if ttl > 0 {
			t := now.Add(time.Duration(ttl) * time.Second)
			resp.ttl = &t
			set = true
		}

		return
	}

	//Expires Header
	//Date format from RFC-2616, Section 14.21
	expires, err := time.Parse(httpDateFormat, resp.Header.Get("Expires"))
	if err != nil {
		return
	}

	if expires.Sub(now) > 0 {
		resp.ttl = &expires
		set = true
	}

	return
}

func setLastModified(resp *Response) bool {
	lastModified, err := time.Parse(httpDateFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return false
	}

	resp.lastModified = &lastModified
	return true
}

func setETag(resp *Response) bool {

	resp.etag = resp.Header.Get("ETag")

	if resp.etag != "" {
		return true
	}

	return false
}
