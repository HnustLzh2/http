package main

import "strings"

// Request 表示一个简单的 HTTP 请求（不依赖 net/http）
type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    string
}

// HandlerResult 表示路由处理函数的结果
type HandlerResult struct {
	Response string // 已经拼好的原始 HTTP 响应报文
	Close    bool   // 是否在发送完响应后关闭连接
}

// HandlerFunc 路由处理函数类型
type HandlerFunc func(req *Request) HandlerResult

// Mux 非 net/http 版本的极简路由器
type Mux struct {
	routes map[string]HandlerFunc
}

// NewMux 创建一个新的路由器
func NewMux() *Mux {
	return &Mux{
		routes: make(map[string]HandlerFunc),
	}
}

// Handle 注册路由
// route 这里约定为 path 的第一个片段，例如：
//   "/"          -> ""
//   "/echo/xxx"  -> "echo"
//   "/user-agent"-> "user-agent"
//   "/files/xx"  -> "files"
func (m *Mux) Handle(route string, handler HandlerFunc) {
	m.routes[route] = handler
}

// Serve 根据 routeKey 分发到对应的 Handler
// 如果没有匹配的路由，则返回 404
func (m *Mux) Serve(routeKey string, req *Request) HandlerResult {
	if h, ok := m.routes[routeKey]; ok {
		return h(req)
	}

	// 默认 404
	closeConn := false
	if conn := req.Headers["Connection"]; conn != "" && strings.ToLower(conn) == "close" {
		closeConn = true
	}

	response := req.Version + " 404 Not Found" + CRLF
	if closeConn {
		response += "Connection: close" + CRLF
	}
	response += CRLF

	return HandlerResult{
		Response: response,
		Close:    closeConn,
	}
}
