package mars

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanicInAction(t *testing.T) {
	startFakeBookingApp()
	TRACE = log.New(ioutil.Discard, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE
	DevMode = false

	ts := httptest.NewServer(Handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/boom")
	if err != nil {
		log.Fatal(err)
	}
	resp, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if !strings.Contains(string(resp), "OMG") {
		t.Error("Unable to get panic description, got:\n", resp)
	}
}

func containsAll(raw []byte, list ...string) bool {
	s := string(raw)
	for _, i := range list {
		if !strings.Contains(s, i) {
			return false
		}
	}

	return true
}

func TestPanicInDevMode(t *testing.T) {
	startFakeBookingApp()
	TRACE = log.New(ioutil.Discard, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE
	DevMode = true

	ts := httptest.NewServer(Handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/boom")
	if err != nil {
		log.Fatal(err)
	}
	resp, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if !containsAll(resp,
		"github.com/roblillack/mars/fakeapp_test.go",
		"github.com/roblillack/mars.Hotels.Boom",
		"OMG") {
		t.Error("Unable to get full panic info, got:\n", string(resp))
	}
}
