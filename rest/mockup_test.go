package rest

import (
	"net/http"
	"testing"
)

func TestMockup(t *testing.T) {

	defer StopMockupServer()
	StartMockupServer()

	myURL := "http://mytest.com/foo?val1=1&val2=2#fragment"

	myHeaders := make(http.Header)
	myHeaders.Add("Hello", "world")

	mock := Mock{
		URL:          myURL,
		HTTPMethod:   http.MethodGet,
		ReqHeaders:   myHeaders,
		RespHTTPCode: http.StatusOK,
		RespBody:     "foo",
	}

	AddMockups(&mock)

	v := Get(myURL)
	if v.String() != "foo" {
		t.Fatal("Mockup Fail!")
	}

}
