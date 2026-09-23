package pconf

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestEnsureOpenGeminiDatabases(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		gotQuery = request.URL.Query()
		_ = request.ParseForm()
		gotForm = request.PostForm
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := EnsureOpenGeminiDatabases(Pushgw{
		Writers: []WriterOptions{{
			Url:     server.URL + "/api/v1/write?db=prom",
			Timeout: 1000,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/query" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotQuery.Get("db") != "prom" {
		t.Fatalf("unexpected database: %q", gotQuery.Get("db"))
	}
	if gotQuery.Get("q") != "" {
		t.Fatalf("unexpected query parameter: %q", gotQuery.Get("q"))
	}
	if gotForm.Get("q") != `CREATE DATABASE "prom"` {
		t.Fatalf("unexpected form query: %q", gotForm.Get("q"))
	}
}

func TestEnsureOpenGeminiDatabasesIgnoresOtherWriters(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	err := EnsureOpenGeminiDatabases(Pushgw{
		Writers: []WriterOptions{{Url: server.URL + "/api/v1/prom/write"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("non-OpenGemini writer was contacted")
	}
}

func TestEnsureOpenGeminiDatabasesRequiresDatabase(t *testing.T) {
	err := EnsureOpenGeminiDatabases(Pushgw{
		Writers: []WriterOptions{{Url: "http://127.0.0.1:8086/api/v1/write"}},
	})
	if err == nil || !strings.Contains(err.Error(), "db query parameter") {
		t.Fatalf("expected missing database error, got %v", err)
	}
}
