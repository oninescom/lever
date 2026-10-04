package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	if err := NewCurlCmd().ExecuteArgs([]string{"-f", "-o", output, server.URL}); err == nil {
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

func TestCurlHeadersUploadAndAuthentication(t *testing.T) {
	var gotMethod, gotBody, gotHeader, gotAgent, gotUser, gotPassword string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotBody = r.Method, string(body)
		gotHeader, gotAgent = r.Header.Get("X-Test"), r.UserAgent()
		gotUser, gotPassword, _ = r.BasicAuth()
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	dir := t.TempDir()
	upload := filepath.Join(dir, "upload.txt")
	headers := filepath.Join(dir, "headers.txt")
	output := filepath.Join(dir, "response.txt")
	if err := os.WriteFile(upload, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(headers, []byte("X-Test: from-file\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := NewCurlCmd().ExecuteArgs([]string{"-T", upload, "-H", "@" + headers, "-u", "alice:secret", "-A", "TestAgent", "-o", output, server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "PUT" || gotBody != "payload" || gotHeader != "from-file" || gotAgent != "TestAgent" || gotUser != "alice" || gotPassword != "secret" {
		t.Fatalf("request = %q %q %q %q %q %q", gotMethod, gotBody, gotHeader, gotAgent, gotUser, gotPassword)
	}
}

func TestCurlHeadAndHTTPErrorModes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "visible")
		w.WriteHeader(http.StatusNotFound)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte("missing"))
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	output := filepath.Join(dir, "response.txt")
	if err := NewCurlCmd().ExecuteArgs([]string{"-I", "-o", output, server.URL}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || !strings.Contains(string(data), "404 Not Found") || !strings.Contains(string(data), "X-Test: visible") || strings.Contains(string(data), "missing") {
		t.Fatalf("HEAD output = %q, %v", data, err)
	}
	if err := NewCurlCmd().ExecuteArgs([]string{"-o", output, server.URL}); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(output)
	if err != nil || string(data) != "missing" {
		t.Fatalf("HTTP error body = %q, %v", data, err)
	}
	if err := NewCurlCmd().ExecuteArgs([]string{"-f", "-o", output, server.URL}); err == nil {
		t.Fatal("expected -f to fail")
	}
	data, err = os.ReadFile(output)
	if err != nil || string(data) != "missing" {
		t.Fatalf("-f changed output = %q, %v", data, err)
	}
}

func TestCurlCLIOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}
		w.Header().Set("X-Test", "visible")
		_, _ = w.Write([]byte("body"))
	}))
	defer server.Close()
	dir := t.TempDir()
	exe := filepath.Join(dir, "lever.exe")
	if output, err := exec.Command("go", "build", "-o", exe, "..").CombinedOutput(); err != nil {
		t.Fatalf("build lever: %v\n%s", err, output)
	}
	run := func(args ...string) (string, string, error) {
		command := exec.Command(exe, append([]string{"curl"}, args...)...)
		command.Dir = dir
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		return stdout.String(), stderr.String(), err
	}
	stdout, _, err := run("-i", server.URL+"/remote.txt")
	if err != nil || !strings.Contains(stdout, "X-Test: visible\r\n") || !strings.HasSuffix(stdout, "\r\nbody") {
		t.Fatalf("-i output = %q, %v", stdout, err)
	}
	stdout, stderr, err := run("-s", "-O", server.URL+"/remote.txt")
	if err != nil || stdout != "" || stderr != "" {
		t.Fatalf("-s -O output = %q %q, %v", stdout, stderr, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "remote.txt"))
	if err != nil || string(data) != "body" {
		t.Fatalf("-O file = %q, %v", data, err)
	}
	stdout, stderr, err = run("-s", "-f", server.URL+"/missing")
	if err == nil || stdout != "" || stderr != "" {
		t.Fatalf("-s -f output = %q %q, %v", stdout, stderr, err)
	}
	stdout, _, err = run("-V")
	if err != nil || !strings.HasPrefix(stdout, "lever curl ") {
		t.Fatalf("-V output = %q, %v", stdout, err)
	}
	stdout, _, err = run("-h", "header")
	if err != nil || !strings.Contains(stdout, "--header") {
		t.Fatalf("-h output = %q, %v", stdout, err)
	}
	_, stderr, err = run("-v", "-o", filepath.Join(dir, "verbose.txt"), server.URL+"/remote.txt")
	if err != nil || !strings.Contains(stderr, "> GET /remote.txt") || !strings.Contains(stderr, "< HTTP/1.1 200 OK") {
		t.Fatalf("-v output = %q, %v", stderr, err)
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
