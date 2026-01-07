package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// CRLF \r\n 是两个字符组成的序列：
// \r：carriage return，中文通常叫 回车
// \n：line feed，中文通常叫 换行
var CRLF = "\r\n" // 回车换行

// 用于存储 --directory 传入的目录
var baseDir string

// 全局路由器
var mux *Mux

func main() {
	// 解析命令行参数，获取 --directory 传入的目录
	// 示例：./your_program.sh --directory /tmp/data/...
	if len(os.Args) >= 3 && os.Args[1] == "--directory" {
		baseDir = os.Args[2]
	}

	// 初始化并注册路由
	mux = NewMux()
	registerRoutes(mux)

	// 启动 HTTP 服务器
	if err := startServer("0.0.0.0:4221"); err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
		os.Exit(1)
	}
}

// startServer 启动 HTTP 服务器，展示 net.Listener 的完整使用模式
func startServer(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("绑定端口失败: %w", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Printf("关闭监听器时出错: %v\n", err)
		} else {
			fmt.Println("监听器已关闭")
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("监听器已关闭，停止接受新连接")
				return nil
			}
			fmt.Printf("接受连接时出错: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// 对同一个 TCP 连接复用同一个 reader，顺序处理多个请求
	reader := bufio.NewReader(conn)
	for {
		// 读取并解析一个请求
		method, paths, version, requestHeaders, body, err := AnalysisRequest(reader)
		if err != nil {
			// 客户端正常关闭连接 → 直接结束循环
			if err == io.EOF {
				return
			}
			// 其他解析错误，打印后结束该连接
			fmt.Println("Error parsing request:", err.Error())
			return
		}

		// 计算用于路由的第一个 path 片段
		splits := strings.Split(paths, "/")
		routeKey := ""
		if len(splits) > 1 {
			routeKey = splits[1]
		}

		req := &Request{
			Method:  method,
			Path:    paths,
			Version: version,
			Headers: requestHeaders,
			Body:    body,
		}

		// 交给 Mux 分发
		result := mux.Serve(routeKey, req)

		// 发送响应
		_, err = conn.Write([]byte(result.Response))
		if err != nil {
			// 写失败一般意味着客户端断开，结束该连接
			return
		}
		if result.Close {
			return // 客户端要求关闭连接，结束该连接
		}
	}
}

// registerRoutes 注册所有路由到 Mux
func registerRoutes(m *Mux) {
	// 根路径 "/"
	m.Handle("", rootHandler)
	// /echo/*
	m.Handle("echo", echoHandler)
	// /user-agent
	m.Handle("user-agent", userAgentHandler)
	// /files/*
	m.Handle("files", filesHandler)
}

func AnalysisRequest(reader *bufio.Reader) (string, string, string, map[string]string, string, error) {
	var method string
	var path string
	var httpVersion string
	headers := make(map[string]string)
	var body string
	line, err := reader.ReadString('\n')
	if err != nil {
		// 直接把错误返回给调用方（包括 EOF）
		return "", "", "", nil, "", err
	}
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return "", "", "", nil, "", fmt.Errorf("invalid request line")
	}
	method = parts[0]
	path = parts[1]
	httpVersion = parts[2]
	// 读取请求头
	for {
		line, err := reader.ReadString('\n') // 读取之后reader会往下走
		if err != nil {
			return "", "", "", nil, "", err
		}
		// 读到尾
		if line == CRLF || line == "\n" {
			break
		}
		line = strings.Trim(line, CRLF) // 去掉首尾的CRLF
		// 按第一个 ':' 分成两部分：key 和 value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value // 设置请求头
	}

	// 读取请求体（如果有 Content-Length）
	if clStr, ok := headers["Content-Length"]; ok {
		length, err := strconv.Atoi(clStr)
		if err != nil {
			return "", "", "", nil, "", fmt.Errorf("invalid Content-Length: %w", err)
		}
		if length > 0 {
			bodyBytes := make([]byte, length)
			_, err = io.ReadFull(reader, bodyBytes)
			if err != nil {
				return "", "", "", nil, "", fmt.Errorf("error reading request body: %w", err)
			}
			body = string(bodyBytes)
		}
	}
	return method, path, httpVersion, headers, body, nil
}
