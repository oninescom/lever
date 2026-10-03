package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCurlRejectsHTTPErrorWithoutOverwritingOutput(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	output := filepath.Join(t.TempDir(), "download.txt")
	if err := os.WriteFile(output, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := NewCurlCmd().ExecuteArgs([]string{"-o", output, server.URL}); err == nil {
		t.Fatal("expected an error for HTTP 404")
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "original" {
		t.Fatalf("output changed on HTTP error: %q, %v", data, err)
	}
}

func TestCurlDownloadsSuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()
	output := filepath.Join(t.TempDir(), "download.txt")
	if err := NewCurlCmd().ExecuteArgs([]string{"-o", output, server.URL}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "downloaded" {
		t.Fatalf("output = %q, %v", data, err)
	}
}

func TestCurlFollowsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("redirected"))
	}))
	defer server.Close()
	output := filepath.Join(t.TempDir(), "redirect.txt")
	if err := NewCurlCmd().ExecuteArgs([]string{"-o", output, server.URL + "/redirect"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "redirected" {
		t.Fatalf("redirect output = %q, %v", data, err)
	}
}

func TestCurlDownloadsLargeResponse(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 2<<20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	output := filepath.Join(t.TempDir(), "large.bin")
	if err := NewCurlCmd().ExecuteArgs([]string{"-o", output, server.URL}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(data, payload) {
		t.Fatalf("large download: %d bytes, %v", len(data), err)
	}
}

func TestCurlRequestMethodAndData(t *testing.T) {
	tests := []struct {
		name, method, body, contentType string
		args                            []string
	}{
		{name: "default GET", method: "GET"},
		{name: "data implies POST", args: []string{"-d", "name=lever"}, method: "POST", body: "name=lever", contentType: "application/x-www-form-urlencoded"},
		{name: "explicit POST with empty data", args: []string{"-X", "post", "-d", ""}, method: "POST", contentType: "application/x-www-form-urlencoded"},
		{name: "explicit GET with data", args: []string{"-X", "GET", "-d", "x=1"}, method: "GET", body: "x=1", contentType: "application/x-www-form-urlencoded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotBody, gotContentType, gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				gotMethod, gotBody, gotContentType, gotPath = r.Method, string(body), r.Header.Get("Content-Type"), r.URL.RequestURI()
				_, _ = w.Write([]byte("ok"))
			}))
			defer server.Close()
			output := filepath.Join(t.TempDir(), "response.txt")
			args := append(append([]string{}, tt.args...), "-o", output, server.URL+"/api?x=2")
			if err := NewCurlCmd().ExecuteArgs(args); err != nil {
				t.Fatal(err)
			}
			if gotMethod != tt.method || gotBody != tt.body || gotContentType != tt.contentType || gotPath != "/api?x=2" {
				t.Fatalf("request = %q %q %q %q; want %q %q %q %q", gotMethod, gotBody, gotContentType, gotPath, tt.method, tt.body, tt.contentType, "/api?x=2")
			}
			data, err := os.ReadFile(output)
			if err != nil || string(data) != "ok" {
				t.Fatalf("response = %q, %v", data, err)
			}
		})
	}
}

func TestWcNewlineCountAndMissingFile(t *testing.T) {
	lines, words, bytes, err := processWcStream(strings.NewReader("one two"))
	if err != nil || lines != 0 || words != 2 || bytes != 7 {
		t.Fatalf("counts = %d %d %d, %v; want 0 2 7", lines, words, bytes, err)
	}
	if err := NewWcCmd().ExecuteArgs([]string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestSedEscapedSlash(t *testing.T) {
	pattern, replacement, global, err := parseSedExpression(`s/a\/b/c\/d/g`)
	if err != nil || pattern != "a/b" || replacement != "c/d" || !global {
		t.Fatalf("parsed %q %q %v, %v", pattern, replacement, global, err)
	}
	if _, _, _, err := parseSedExpression("s/a/b/invalid"); err == nil {
		t.Fatal("expected an error for an unsupported suffix")
	}
}

func TestPingRejectsInvalidInterval(t *testing.T) {
	if err := NewPingCmd().ExecuteArgs([]string{"-i", "0", "127.0.0.1"}); err == nil {
		t.Fatal("expected an error for zero interval")
	}
}
