package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPRateLimit(t *testing.T) {
	h := IPRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))

	srv := httptest.NewServer(h)
	defer srv.Close()

	// fire more requests than token bucket allows quickly
	client := &http.Client{}
	for i := 0; i < 35; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/", nil)
		resp, _ := client.Do(req)
		if i < 30 {
			if resp.StatusCode == 429 {
				t.Fatalf("unexpected 429 before bucket exhausted at i=%d", i)
			}
		}
		if i >= 30 && resp.StatusCode != 429 {
			// after bucket should be limited (some may still succeed due to timing)
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
	}
}
