//go:build go1.18
// +build go1.18

package mars

//go

import (
	"fmt"
	"net/http"
	"testing"
)

func makeCookie(args Args) string {
	session := make(Session)
	session.SetDefaultExpiration()
	for k, v := range args {
		session[k] = fmt.Sprint(v)
	}
	return session.Cookie().Value
}

func FuzzSessionDecoding(f *testing.F) {
	secretKey = generateRandomSecretKey()

	f.Add(makeCookie(Args{"username": "roblillack"}))
	f.Add(makeCookie(Args{"username": "roblillack", "lang": "de"}))
	f.Add(makeCookie(Args{"username": "roblillack", "lang": "de", "orientation": "portrait"}))
	f.Add(makeCookie(Args{"username": "roblillack", "bw": true}))
	f.Add(makeCookie(Args{"no": 28963473, "bw": true}))

	f.Fuzz(func(t *testing.T, cookieContent string) {
		cookie := &http.Cookie{Value: cookieContent}
		if session := GetSessionFromCookie(cookie); session == nil {
			t.Fail()
		}
	})
}
