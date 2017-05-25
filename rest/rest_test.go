package rest

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	resp := Get(server.URL + "/user")
	if resp.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200)")
	}

}

func TestSlowGet(t *testing.T) {

	var f [100]*Response

	for i := range f {
		f[i] = rb.Get("/slow/user")

		if f[i].Response.StatusCode != http.StatusOK {
			t.Fatal("f Status != OK (200)")
		}
	}
}

func TestHead(t *testing.T) {
	resp := Head(server.URL + "/user")

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200)")
	}

}

func TestPost(t *testing.T) {
	resp := Post(server.URL+"/user", &User{Name: "Matilda"})

	if resp.StatusCode != http.StatusCreated {
		t.Fatal("Status != OK (201)")
	}
}

func TestPostXML(t *testing.T) {

	rbXML := RequestBuilder{
		BaseURL:     server.URL,
		ContentType: XML,
	}

	resp := rbXML.Post("/xml/user", &User{Name: "Matilda"})

	if resp.StatusCode != http.StatusCreated {
		t.Fatal("Status != OK (201)")
	}
}

func TestPut(t *testing.T) {
	resp := Put(server.URL+"/user/3", &User{Name: "Pichucha"})

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200")
	}
}

func TestPatch(t *testing.T) {
	resp := Patch(server.URL+"/user/3", &User{Name: "Pichucha"})

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200")
	}
}

func TestDelete(t *testing.T) {
	resp := Delete(server.URL + "/user/4")

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200")
	}
}

func TestOptions(t *testing.T) {
	resp := Options(server.URL + "/user")

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200")
	}
}

func TestAsyncGet(t *testing.T) {

	AsyncGet(server.URL+"/user", func(r *Response) {
		if r.StatusCode != http.StatusOK {
			t.Fatal("Status != OK (200)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestAsyncHead(t *testing.T) {

	AsyncHead(server.URL+"/user", func(r *Response) {
		if r.StatusCode != http.StatusOK {
			t.Fatal("Status != OK (200)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestAsyncPost(t *testing.T) {

	AsyncPost(server.URL+"/user", &User{Name: "Matilda"}, func(r *Response) {
		if r.StatusCode != http.StatusCreated {
			t.Fatal("Status != OK (201)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestAsyncPut(t *testing.T) {

	AsyncPut(server.URL+"/user/3", &User{Name: "Pichucha"}, func(r *Response) {
		if r.StatusCode != http.StatusOK {
			t.Fatal("Status != OK (200)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestAsyncPatch(t *testing.T) {

	AsyncPatch(server.URL+"/user/3", &User{Name: "Pichucha"}, func(r *Response) {
		if r.StatusCode != http.StatusOK {
			t.Fatal("Status != OK (200)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestAsyncDelete(t *testing.T) {

	AsyncDelete(server.URL+"/user/4", func(r *Response) {
		if r.StatusCode != http.StatusOK {
			t.Fatal("Status != OK (200)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestAsyncOptions(t *testing.T) {

	AsyncOptions(server.URL+"/user", func(r *Response) {
		if r.StatusCode != http.StatusOK {
			t.Fatal("Status != OK (200)")
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestHeaders(t *testing.T) {

	h := make(http.Header)
	h.Add("X-Test", "test")

	builder := RequestBuilder{
		BaseURL: server.URL,
		Headers: h,
	}

	r := builder.Get("/header")

	if r.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200)")
	}

}

func TestWrongURL(t *testing.T) {
	r := Get("foo")
	if r.Err == nil {
		t.Fatal("Wrong URL should get an error")
	}
}

/*Increase percentage of net.go coverage */
func TestRequestWithProxyAndFollowRedirect(t *testing.T) {

	customPool := CustomPool{
		MaxIdleConnsPerHost: 100,
		Proxy:               "http://saraza",
	}

	restClient := new(RequestBuilder)
	restClient.ContentType = JSON
	restClient.DisableTimeout = true
	restClient.CustomPool = &customPool
	restClient.FollowRedirect = true

	response := restClient.Get(server.URL + "/user")
	expected := "error connecting to proxy"

	if !strings.Contains(response.Err.Error(), expected) {
		t.Fatalf("Expected %v Error, Got %v as Response", expected, response.Err.Error())
	}
}

func TestRequestSendingClientMetrics(t *testing.T) {

	restClient := new(RequestBuilder)

	response := restClient.Get(server.URL + "/user")

	if response.StatusCode != http.StatusOK {
		t.Fatal("Status != OK (200)")
	}
}

func TestResponseExceedsConnectTimeout(t *testing.T) {

	restClient := RequestBuilder{CustomPool: &CustomPool{}}
	restClient.ConnectTimeout = 1 * time.Nanosecond
	restClient.Timeout = 35 * time.Millisecond
	restClient.ContentType = JSON

	scuResponse := restClient.Get(server.URL + "/cache/slow/user")

	scuResponseErrIsTimeoutExceeded := func() bool {
		expected := "dial tcp"
		if scuResponse.Err != nil {
			return strings.Contains(scuResponse.Err.Error(), expected)
		}
		return false
	}

	if !scuResponseErrIsTimeoutExceeded() {
		t.Errorf("Timeouts configuration should get an error when connect")
	}
}

func TestResponseExceedsRequestTimeout(t *testing.T) {

	restClient := RequestBuilder{CustomPool: &CustomPool{Transport: &http.Transport{}}}
	restClient.ConnectTimeout = 35 * time.Millisecond
	restClient.Timeout = 9 * time.Millisecond
	restClient.ContentType = JSON

	suResponse := restClient.Get(server.URL + "/slow/user")

	suResponseErrIsTimeoutExceeded := func() bool {
		expected := "timeout awaiting response headers"
		if suResponse.Err != nil {
			return strings.Contains(suResponse.Err.Error(), expected)
		}
		return false
	}

	if !suResponseErrIsTimeoutExceeded() {
		t.Fatalf("Timeouts configuration should get an error after connect")
	}
}
