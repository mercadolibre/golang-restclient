package rest

import (
	"net/http"
	"strconv"
	"testing"
)

func TestForkJoin(t *testing.T) {

	var f [7]*FutureResponse
	var post *FutureResponse

	ForkJoin(func(cr *Concurrent) {
		f[0] = cr.Get(server.URL + "/user/1")
		f[1] = cr.Get(server.URL + "/user/2")
		f[2] = cr.Head(server.URL + "/user")
		f[3] = cr.Put(server.URL+"/user/3", &User{Name: "Pichucha"})
		f[4] = cr.Delete(server.URL + "/user/4")
		f[5] = cr.Patch(server.URL+"/user/3", &User{Name: "Pichucha"})
		f[6] = cr.Options(server.URL + "/user")

		post = cr.Post(server.URL+"/user", &User{Name: "Matilda"})
	})

	for i := range f {
		if f[i].Response().StatusCode != http.StatusOK {
			if f[i].Response().Err != nil {
				t.Fatal(f[i].Response().Err)
			} else {
				t.Fatal("f1 Status != OK (200). Status = " + strconv.Itoa(f[i].Response().StatusCode))
			}
		}
	}

	if post.Response().StatusCode != http.StatusCreated {
		t.Fatal("f2 Status != OK (201)")
	}

}

func TestSlowForkJoinGet(t *testing.T) {

	var f [100]*FutureResponse

	for x := 0; x < 50; x++ {

		rb.ForkJoin(func(cr *Concurrent) {
			for i := range f {
				f[i] = cr.Get("/slow/user")
			}
		})

		for i := range f {
			if f[i].Response().StatusCode != http.StatusOK {
				t.Fatal("f[" + strconv.Itoa(i) + "] Status != OK (200)")
			}
		}

	}

}
