package commands

import (
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var winInetDLL = windows.NewLazySystemDLL("wininet.dll")
var internetOpen = winInetDLL.NewProc("InternetOpenW")
var internetConnect = winInetDLL.NewProc("InternetConnectW")
var httpOpenRequest = winInetDLL.NewProc("HttpOpenRequestW")
var httpSendRequest = winInetDLL.NewProc("HttpSendRequestW")
var internetReadFile = winInetDLL.NewProc("InternetReadFile")
var internetCloseHandle = winInetDLL.NewProc("InternetCloseHandle")
var httpQueryInfo = winInetDLL.NewProc("HttpQueryInfoW")

func NewCurlCmd() *engine.Command {
	var outputFile string
	var method string
	var data string
	cmd := &engine.Command{
		Use:   "curl [URL]",
		Short: "Send an HTTP request or download its response",
		Long:  `Fetch an HTTP or HTTPS URL. Use -X to set the method, -d to send form data (POST by default), and -o to save the response to a file. Otherwise the response goes to standard output.`,
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			requestMethod := strings.ToUpper(method)
			hasData := c.Flags().Changed("data")
			if hasData && !c.Flags().Changed("request") {
				requestMethod = "POST"
			}
			return fetchURL(args[0], requestMethod, data, hasData, outputFile)
		},
	}
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write the response to a file")
	cmd.Flags().StringVarP(&method, "request", "X", "GET", "Set the HTTP request method")
	cmd.Flags().StringVarP(&data, "data", "d", "", "Send form data in the request body (implies POST unless -X is set)")
	return cmd
}

func fetchURL(rawURL, method, data string, hasData bool, outputFile string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("curl error: invalid URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return fmt.Errorf("curl error: URL must start with http:// or https://")
	}
	if parsed.User != nil {
		return fmt.Errorf("curl error: URL credentials are not supported")
	}
	port := 80
	if parsed.Scheme == "https" {
		port = 443
	}
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("curl error: invalid URL port")
		}
	}
	object := parsed.EscapedPath()
	if object == "" {
		object = "/"
	}
	if parsed.ForceQuery || parsed.RawQuery != "" {
		object += "?" + parsed.RawQuery
	}
	agent := windows.StringToUTF16Ptr("Lever")
	host, err := windows.UTF16PtrFromString(parsed.Hostname())
	if err != nil {
		return fmt.Errorf("curl error: invalid URL: %w", err)
	}
	verb, err := windows.UTF16PtrFromString(method)
	if err != nil {
		return fmt.Errorf("curl error: invalid method: %w", err)
	}
	path, err := windows.UTF16PtrFromString(object)
	if err != nil {
		return fmt.Errorf("curl error: invalid URL: %w", err)
	}
	session, _, callErr := internetOpen.Call(uintptr(unsafe.Pointer(agent)), 0, 0, 0, 0)
	if session == 0 {
		return fmt.Errorf("curl error: cannot open internet session: %w", callErr)
	}
	defer internetCloseHandle.Call(session)
	const internetServiceHTTP = 3
	connection, _, callErr := internetConnect.Call(session, uintptr(unsafe.Pointer(host)), uintptr(port), 0, 0, internetServiceHTTP, 0, 0)
	runtime.KeepAlive(host)
	if connection == 0 {
		return fmt.Errorf("curl error: cannot connect to host: %w", callErr)
	}
	defer internetCloseHandle.Call(connection)
	// Reload the resource and avoid persisting response data in the WinINet cache.
	flags := uintptr(0x80000000 | 0x04000000)
	if parsed.Scheme == "https" {
		flags |= 0x00800000 // INTERNET_FLAG_SECURE
	}
	request, _, callErr := httpOpenRequest.Call(connection, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(path)), 0, 0, 0, flags, 0)
	runtime.KeepAlive(verb)
	runtime.KeepAlive(path)
	if request == 0 {
		return fmt.Errorf("curl error: cannot open HTTP request: %w", callErr)
	}
	defer internetCloseHandle.Call(request)
	var headerPtr uintptr
	var headerLength uintptr
	var header *uint16
	if hasData {
		header = windows.StringToUTF16Ptr("Content-Type: application/x-www-form-urlencoded\r\n")
		headerPtr = uintptr(unsafe.Pointer(header))
		headerLength = uintptr(len("Content-Type: application/x-www-form-urlencoded\r\n"))
	}
	body := []byte(data)
	var bodyPtr uintptr
	if len(body) > 0 {
		bodyPtr = uintptr(unsafe.Pointer(&body[0]))
	}
	ok, _, callErr := httpSendRequest.Call(request, headerPtr, headerLength, bodyPtr, uintptr(len(body)))
	runtime.KeepAlive(header)
	runtime.KeepAlive(body)
	if ok == 0 {
		return fmt.Errorf("curl error: network request failed: %w", callErr)
	}

	var status uint32
	statusSize := uint32(unsafe.Sizeof(status))
	const statusCodeQuery = 19 | 0x20000000 // HTTP_QUERY_STATUS_CODE | HTTP_QUERY_FLAG_NUMBER
	ok, _, callErr = httpQueryInfo.Call(request, statusCodeQuery, uintptr(unsafe.Pointer(&status)), uintptr(unsafe.Pointer(&statusSize)), 0)
	if ok == 0 {
		return fmt.Errorf("curl error: cannot read HTTP status: %w", callErr)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("curl error: HTTP request failed: status %d", status)
	}

	var writer io.Writer = os.Stdout
	var file *os.File
	if outputFile != "" {
		file, err = os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("curl error: cannot create destination file: %w", err)
		}
		defer file.Close()
		writer = file
		fmt.Fprintf(os.Stderr, "Downloading %s to file '%s'...\n", rawURL, outputFile)
	}
	buffer := make([]byte, 32768)
	for {
		var read uint32
		ok, _, callErr := internetReadFile.Call(request, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(unsafe.Pointer(&read)))
		if ok == 0 {
			return fmt.Errorf("curl error: cannot read response: %w", callErr)
		}
		if read == 0 {
			break
		}
		if _, err := writer.Write(buffer[:read]); err != nil {
			return fmt.Errorf("curl error: cannot write response: %w", err)
		}
	}
	if file != nil {
		if err := file.Close(); err != nil {
			return fmt.Errorf("curl error: cannot close destination file: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Download completed.")
	}
	return nil
}
