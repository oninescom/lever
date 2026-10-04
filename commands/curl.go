package commands

import (
	"encoding/base64"
	"fmt"
	"io"
	"lever/engine"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type curlOptions struct {
	method, data, output, upload, user, userAgent                 string
	headers                                                       []string
	fail, head, remoteName, showHeaders, silent, verbose, version bool
	hasData                                                       bool
}

// SilentCurlError keeps a nonzero exit status without printing diagnostics for -s.
type SilentCurlError struct{ Err error }

func (e *SilentCurlError) Error() string { return e.Err.Error() }
func (e *SilentCurlError) Unwrap() error { return e.Err }

func NewCurlCmd() *engine.Command {
	var opts curlOptions
	cmd := &engine.Command{
		Use:   "curl [options...] <url>",
		Short: "Send an HTTP request or download its response",
		Long:  "Fetch an HTTP or HTTPS URL and write the response to standard output or a file.",
		Args: func(args []string) error {
			if opts.version {
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("curl requires exactly one URL")
			}
			return nil
		},
		RunE: func(c *engine.Command, args []string) error {
			if opts.version {
				fmt.Fprintln(os.Stdout, "lever curl 1.0.0")
				return nil
			}
			opts.hasData = c.Flags().Changed("data")
			err := fetchURL(args[0], opts)
			if err != nil && opts.silent {
				return &SilentCurlError{Err: err}
			}
			return err
		},
	}
	f := cmd.Flags()
	f.StringVarP(&opts.data, "data", "d", "", "HTTP POST data")
	f.BoolVarP(&opts.fail, "fail", "f", false, "Fail fast with no output on HTTP errors")
	f.BoolVarP(&opts.head, "head", "I", false, "Show document info only")
	f.StringArrayVarP(&opts.headers, "header", "H", nil, "Pass custom header(s), or @file, to server")
	f.StringVarP(&opts.output, "output", "o", "", "Write to file instead of stdout")
	f.BoolVarP(&opts.remoteName, "remote-name", "O", false, "Write output to file named as remote file")
	f.BoolVarP(&opts.showHeaders, "show-headers", "i", false, "Show response headers in output")
	f.BoolVarP(&opts.silent, "silent", "s", false, "Silent mode")
	f.StringVarP(&opts.upload, "upload-file", "T", "", "Transfer local file to destination")
	f.StringVarP(&opts.user, "user", "u", "", "Server user and password")
	f.StringVarP(&opts.userAgent, "user-agent", "A", "Lever", "Send User-Agent to server")
	f.BoolVarP(&opts.verbose, "verbose", "v", false, "Make the operation more talkative")
	f.BoolVarP(&opts.version, "version", "V", false, "Show version number and quit")
	f.StringVarP(&opts.method, "request", "X", "", "Set the HTTP request method")
	return cmd
}

func fetchURL(rawURL string, opts curlOptions) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return fmt.Errorf("curl error: URL must start with http:// or https://")
	}
	if parsed.User != nil {
		return fmt.Errorf("curl error: URL credentials are not supported; use -u")
	}
	if opts.hasData && opts.upload != "" {
		return fmt.Errorf("curl error: -d and -T cannot be combined")
	}
	if opts.output != "" && opts.remoteName {
		return fmt.Errorf("curl error: -o and -O cannot be combined")
	}
	method := "GET"
	if opts.hasData {
		method = "POST"
	}
	if opts.upload != "" {
		method = "PUT"
	}
	if opts.head {
		method = "HEAD"
	}
	if opts.method != "" {
		method = strings.ToUpper(opts.method)
	}
	var body io.Reader
	if opts.hasData {
		body = strings.NewReader(opts.data)
	}
	if opts.upload != "" {
		file, err := os.Open(opts.upload)
		if err != nil {
			return fmt.Errorf("curl error: cannot open upload file: %w", err)
		}
		defer file.Close()
		body = file
	}
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return fmt.Errorf("curl error: invalid request: %w", err)
	}
	req.Header.Set("User-Agent", opts.userAgent)
	if opts.hasData {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if opts.user != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(opts.user)))
	}
	for _, value := range opts.headers {
		if after, ok := strings.CutPrefix(value, "@"); ok {
			data, err := os.ReadFile(after)
			if err != nil {
				return fmt.Errorf("curl error: cannot read header file: %w", err)
			}
			value = string(data)
		}
		for line := range strings.SplitSeq(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
			if line == "" {
				continue
			}
			name, val, ok := strings.Cut(line, ":")
			if !ok || strings.TrimSpace(name) == "" {
				return fmt.Errorf("curl error: invalid header %q", line)
			}
			req.Header.Set(strings.TrimSpace(name), strings.TrimSpace(val))
		}
	}
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", method, req.URL.RequestURI())
		for name, values := range req.Header {
			if strings.EqualFold(name, "Authorization") {
				continue
			}
			for _, value := range values {
				fmt.Fprintf(os.Stderr, "> %s: %s\n", name, value)
			}
		}
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("curl error: network request failed: %w", err)
	}
	defer response.Body.Close()
	if opts.verbose {
		fmt.Fprintln(os.Stderr, "<", response.Proto, response.Status)
	}
	if opts.fail && response.StatusCode >= 400 {
		return fmt.Errorf("curl error: HTTP request failed: status %d", response.StatusCode)
	}
	output := opts.output
	if opts.remoteName {
		if strings.HasSuffix(parsed.Path, "/") || strings.HasSuffix(parsed.Path, "\\") {
			return fmt.Errorf("curl error: URL has no remote filename")
		}
		output = path.Base(strings.ReplaceAll(parsed.Path, "\\", "/"))
		if output == "." || output == "/" || output == "" {
			return fmt.Errorf("curl error: URL has no remote filename")
		}
	}
	var writer io.Writer = os.Stdout
	var file *os.File
	if output != "" {
		file, err = os.Create(output)
		if err != nil {
			return fmt.Errorf("curl error: cannot create destination file: %w", err)
		}
		defer file.Close()
		writer = file
		if !opts.silent {
			fmt.Fprintf(os.Stderr, "Downloading %s to file '%s'...\n", rawURL, output)
		}
	}
	if opts.head || opts.showHeaders {
		if _, err := fmt.Fprintf(writer, "%s %s\r\n", response.Proto, response.Status); err != nil {
			return fmt.Errorf("curl error: cannot write response: %w", err)
		}
		if err := response.Header.Write(writer); err != nil {
			return fmt.Errorf("curl error: cannot write response: %w", err)
		}
		if _, err := io.WriteString(writer, "\r\n"); err != nil {
			return fmt.Errorf("curl error: cannot write response: %w", err)
		}
	}
	if !opts.head {
		progress := newProgressBar("Downloading", response.ContentLength)
		if output == "" || opts.silent {
			progress = nil
		}
		if progress != nil {
			defer progress.Finish()
			writer = io.MultiWriter(writer, progress)
		}
		if _, err := io.Copy(writer, response.Body); err != nil {
			return fmt.Errorf("curl error: cannot read or write response: %w", err)
		}
	}
	if file != nil {
		if err := file.Close(); err != nil {
			return fmt.Errorf("curl error: cannot close destination file: %w", err)
		}
		if !opts.silent {
			fmt.Fprintln(os.Stderr, "Download completed.")
		}
	}
	return nil
}
