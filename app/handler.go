package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 根路径 Handler：返回 200 OK，无 body
func rootHandler(req *Request) HandlerResult {
	closeConn := strings.ToLower(req.Headers["Connection"]) == "close"

	if closeConn {
		// 带 Connection: close 的版本
		headers := "Connection: close" + CRLF
		response := req.Version + " 200 OK" + CRLF + headers + CRLF + CRLF
		return HandlerResult{
			Response: response,
			Close:    true,
		}
	}

	response := req.Version + " 200 OK" + CRLF + CRLF
	return HandlerResult{
		Response: response,
		Close:    false,
	}
}

// /echo/<text> Handler
func echoHandler(req *Request) HandlerResult {
	splits := strings.Split(req.Path, "/")
	if len(splits) < 3 {
		return HandlerResult{
			Response: "HTTP/1.1 400 Bad Request" + CRLF + CRLF,
			Close:    false,
		}
	}
	str := splits[2]

	responseHeaders := make(map[string]string)
	contentType := "text/plain"

	// gzip 压缩协商
	if enc := req.Headers["Accept-Encoding"]; enc != "" {
		compresses := enc
		fmt.Println("compresses: ", compresses)
		compress := strings.Split(compresses, ",")
		fmt.Println("compress: ", compress)
		for _, c := range compress {
			if strings.TrimSpace(c) == "gzip" || strings.TrimSpace(c) == "zip" {
				responseHeaders["Content-Encoding"] = strings.TrimSpace(c)
				break
			}
		}
	}

	// Connection: close
	if strings.ToLower(req.Headers["Connection"]) == "close" {
		responseHeaders["Connection"] = "close"
	}

	// 基础头部
	responseHeaders["Content-Type"] = contentType

	// 判断是否需要压缩
	_, isCompress := responseHeaders["Content-Encoding"]

	var bodyBytes []byte
	if isCompress {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write([]byte(str)); err != nil {
			return HandlerResult{
				Response: req.Version + " 500 Internal Server Error" + CRLF + CRLF,
				Close:    false,
			}
		}
		gw.Close()
		bodyBytes = buf.Bytes()
	} else {
		bodyBytes = []byte(str)
	}

	// Content-Length
	contentLength := len(bodyBytes)
	responseHeaders["Content-Length"] = strconv.Itoa(contentLength)

	// 拼接头部字符串
	var responseHeadersStr string
	for head, value := range responseHeaders {
		responseHeadersStr += head + ": " + value + CRLF
	}

	response := req.Version + " 200 OK" + CRLF + responseHeadersStr + CRLF + string(bodyBytes)

	return HandlerResult{
		Response: response,
		Close:    strings.ToLower(req.Headers["Connection"]) == "close",
	}
}

// /user-agent Handler
func userAgentHandler(req *Request) HandlerResult {
	userAgent := req.Headers["User-Agent"]

	responseHeaders := make(map[string]string)
	contentType := "text/plain"

	responseHeaders["Content-Type"] = contentType
	responseHeaders["Content-Length"] = strconv.Itoa(len(userAgent))
	if strings.ToLower(req.Headers["Connection"]) == "close" {
		responseHeaders["Connection"] = "close"
	}

	var responseHeadersStr string
	for head, value := range responseHeaders {
		responseHeadersStr += head + ": " + value + CRLF
	}

	response := req.Version + " 200 OK" + CRLF + responseHeadersStr + CRLF + userAgent

	return HandlerResult{
		Response: response,
		Close:    strings.ToLower(req.Headers["Connection"]) == "close",
	}
}

// /files/* Handler
func filesHandler(req *Request) HandlerResult {
	splits := strings.Split(req.Path, "/")
	if len(splits) < 3 {
		return HandlerResult{
			Response: req.Version + " 400 Bad Request" + CRLF + CRLF,
			Close:    false,
		}
	}

	// 写文件
	if req.Method == "POST" {
		fileName := strings.Join(splits[2:], "/")
		file, err := os.Create(filepath.Join(baseDir, fileName))
		if err != nil {
			return HandlerResult{
				Response: req.Version + " 500 Internal Server Error" + CRLF + CRLF,
				Close:    false,
			}
		}
		defer file.Close()

		if _, err = file.Write([]byte(req.Body)); err != nil {
			return HandlerResult{
				Response: req.Version + " 500 Internal Server Error" + CRLF + CRLF,
				Close:    false,
			}
		}

		return HandlerResult{
			Response: req.Version + " 201 Created" + CRLF + CRLF,
			Close:    false,
		}
	}

	// 读文件
	relPath := strings.Join(splits[2:], "/")
	filePath := filepath.Join(baseDir, relPath)
	if _, err := os.Stat(filePath); err != nil {
		return HandlerResult{
			Response: req.Version + " 404 Not Found" + CRLF + CRLF,
			Close:    false,
		}
	}
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return HandlerResult{
			Response: req.Version + " 500 Internal Server Error" + CRLF + CRLF,
			Close:    false,
		}
	}

	responseHeaders := make(map[string]string)
	responseHeaders["Content-Type"] = "application/octet-stream"
	responseHeaders["Content-Length"] = strconv.Itoa(len(contentBytes))
	if strings.ToLower(req.Headers["Connection"]) == "close" {
		responseHeaders["Connection"] = "close"
	}

	var responseHeadersStr string
	for head, value := range responseHeaders {
		responseHeadersStr += head + ": " + value + CRLF
	}

	response := req.Version + " 200 OK" + CRLF + responseHeadersStr + CRLF + string(contentBytes)

	return HandlerResult{
		Response: response,
		Close:    strings.ToLower(req.Headers["Connection"]) == "close",
	}
}
