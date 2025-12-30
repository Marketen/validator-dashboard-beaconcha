package validator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/beaconcha"
)

func TestIntegrationHandler(t *testing.T) {
	// Mock Beaconcha server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/ethereum/validators", func(w http.ResponseWriter, r *http.Request) {
		// return a data array with one validator
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"validator":              map[string]interface{}{"index": 1},
					"slashed":                false,
					"status":                 "active",
					"withdrawal_credentials": map[string]interface{}{"type": "bls", "address": "0xabc"},
					"life_cycle_epochs":      map[string]interface{}{"activation": 123, "exit": 0},
					"balances":               map[string]interface{}{"current": "1000000000000000000"},
					"online":                 true,
					"beacon_score":           0.99,
				},
			},
		})
	})
	mux.HandleFunc("/api/v2/ethereum/validators/rewards-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"validator": map[string]interface{}{"index": 1}, "total": "1000000000000000", "total_reward": "1000000000000000", "total_penalty": "0"},
			},
		})
	})
	mux.HandleFunc("/api/v2/ethereum/validators/performance-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"validator": map[string]interface{}{"index": 1}, "beaconscore": map[string]interface{}{"total": 0.99}},
			},
		})
	})

	bs := httptest.NewServer(mux)
	defer bs.Close()

	// replace global client with one pointing to mock server
	bcClient = beaconcha.NewClient(bs.URL)

	// start our handler server
	srv := httptest.NewServer(http.HandlerFunc(Handler))
	defer srv.Close()

	// make request
	body := map[string]interface{}{"validatorId": []int{1}}
	bb, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/validator", "application/json", bytes.NewReader(bb))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
