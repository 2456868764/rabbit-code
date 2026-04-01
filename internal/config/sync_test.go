package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncPullToUserFile_merge(t *testing.T) {
	global := t.TempDir()
	user := filepath.Join(global, UserConfigFileName)
	if err := os.WriteFile(user, []byte(`{"keep":"yes","overlay":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"overlay": "new",
			"fromSrv": float64(1),
		})
	}))
	defer srv.Close()

	if err := SyncPullToUserFile(context.Background(), srv.URL, global); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJSONFile(user)
	if err != nil {
		t.Fatal(err)
	}
	if got["keep"] != "yes" || got["overlay"] != "new" || got["fromSrv"].(float64) != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestSyncPushFromUserFile(t *testing.T) {
	global := t.TempDir()
	user := filepath.Join(global, UserConfigFileName)
	if err := os.WriteFile(user, []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := SyncPushFromUserFile(context.Background(), srv.URL, global); err != nil {
		t.Fatal(err)
	}
	var v map[string]interface{}
	if err := json.Unmarshal(gotBody, &v); err != nil {
		t.Fatal(err)
	}
	if v["x"].(float64) != 1 {
		t.Fatalf("%+v", v)
	}
}
