package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	grpcloader "LiteBlog/utils/plugins/gRPCLoader"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type PluginClient struct {
	conn      *grpc.ClientConn
	client    grpcloader.PluginServiceClient
	id        string
	accessKey string // 访问密钥
}

func NewPluginClient(serverAddr string, accessKey string) (*PluginClient, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	client := grpcloader.NewPluginServiceClient(conn)
	return &PluginClient{
		conn:      conn,
		client:    client,
		accessKey: accessKey,
	}, nil
}

func (c *PluginClient) Initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 添加 access key 到 metadata（如果配置了）
	if c.accessKey != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "access-key", c.accessKey)
	}

	resp, err := c.client.Initialize(ctx, &grpcloader.Empty{})
	if err != nil {
		return fmt.Errorf("initialize failed: %v", err)
	}

	c.id = resp.Id
	log.Printf("Initialized successfully. ID: %s, Version: %s\n", resp.Id, resp.Version)
	return nil
}

func (c *PluginClient) withIDContext(ctx context.Context) context.Context {
	if c.id == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "id", c.id)
}

// RegisterMethods 注册插件方法到服务器，并获取服务器可用的公共方法列表
func (c *PluginClient) RegisterMethods(methods []*grpcloader.MethodDef) error {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 5*time.Second)
	defer cancel()

	resp, err := c.client.RegisterPluginMethods(ctx, &grpcloader.RegisterMethodsRequest{
		Methods: methods,
	})
	if err != nil {
		return err
	}
	// 打印服务器可用的方法列表
	if resp != nil && len(resp.Methods) > 0 {
		log.Printf("Server available methods: %d", len(resp.Methods))
		for _, m := range resp.Methods {
			log.Printf("  - %s", m.Name)
		}
	}
	return nil
}

func (c *PluginClient) Unload() error {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 5*time.Second)
	defer cancel()

	_, err := c.client.UnLoad(ctx, &grpcloader.Verify{Id: c.id})
	return err
}

func (c *PluginClient) CallServerMethod(method string, args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 10*time.Second)
	defer cancel()

	resp, err := c.client.CallServerMethod(ctx, &grpcloader.CallMethod{
		Method: method,
		Args:   args,
	})
	if err != nil {
		return nil, err
	}
	return resp.Args, nil
}

func (c *PluginClient) NewCommandStream() (grpcloader.PluginService_NewCommandStreamClient, error) {
	ctx := c.withIDContext(context.Background())
	return c.client.NewCommandStream(ctx)
}

func (c *PluginClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: client <server-address> [access-key]")
	}
	serverAddr := os.Args[1]
	accessKey := ""
	if len(os.Args) >= 3 {
		accessKey = os.Args[2]
	}

	// 创建客户端
	client, err := NewPluginClient(serverAddr, accessKey)
	if err != nil {
		log.Fatalf("Client creation failed: %v", err)
	}
	defer client.Close()

	// 初始化插件
	if err := client.Initialize(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	// 注册&加载方法
	methods := []*grpcloader.MethodDef{
		{Name: "Example.Callback.onRequest"},
		{Name: "Example.Callback.onUserRequest"},   // 参数化路由处理
		{Name: "Example.Callback.onStaticRequest"}, // 通配符路由处理
		{Name: "Example.Listener.onRoute"},         // 路由监听示例
	}
	if err := client.RegisterMethods(methods); err != nil {
		log.Printf("Method registration failed: %v", err)
	} else {
		log.Println("Methods registered successfully")
	}

	// 创建命令流
	stream, err := client.NewCommandStream()
	if err != nil {
		log.Printf("Command stream creation failed: %v", err)
	} else {
		go handleCommandStream(stream)
	}

	// 示例1: 精确匹配路由
	args := []*grpcloader.Arg{
		{Name: "class", Type: "string", Arg: []byte("onRequest")},
		{Name: "callback", Type: "string", Arg: []byte("Example.Callback.onRequest")},
		{Name: "name", Type: "string", Arg: []byte("/welcome")}, // 精确匹配
	}
	if results, err := client.CallServerMethod("AddHook", args); err != nil {
		log.Printf("AddHook failed: %v", err)
	} else {
		log.Printf("Hook /welcome registered: %+v", results)
	}

	// 示例2: 参数化路由 - 使用 :param 匹配单个路径段
	args = []*grpcloader.Arg{
		{Name: "class", Type: "string", Arg: []byte("onRequest")},
		{Name: "callback", Type: "string", Arg: []byte("Example.Callback.onUserRequest")},
		{Name: "name", Type: "string", Arg: []byte("/api/users/:id")}, // 参数匹配
	}
	if results, err := client.CallServerMethod("AddHook", args); err != nil {
		log.Printf("AddHook failed: %v", err)
	} else {
		log.Printf("Hook /api/users/:id registered: %+v", results)
	}

	// 示例3: 通配符路由 - 使用 *wildcard 匹配剩余所有路径
	args = []*grpcloader.Arg{
		{Name: "class", Type: "string", Arg: []byte("onRequest")},
		{Name: "callback", Type: "string", Arg: []byte("Example.Callback.onStaticRequest")},
		{Name: "name", Type: "string", Arg: []byte("/static/*filepath")}, // 通配符匹配
	}
	if results, err := client.CallServerMethod("AddHook", args); err != nil {
		log.Printf("AddHook failed: %v", err)
	} else {
		log.Printf("Hook /static/*filepath registered: %+v", results)
	}

	log.Println("Routes registered:")
	log.Println("  - GET /welcome           -> onRequest (exact match)")
	log.Println("  - GET /api/users/:id     -> onUserRequest (param match)")
	log.Println("  - GET /static/*filepath  -> onStaticRequest (wildcard match)")

	// 路由监听示例：监听 /welcome 请求/响应
	listenerArgs := []*grpcloader.Arg{
		{Name: "route", Type: "string", Arg: []byte("/welcome")},
		{Name: "callback", Type: "string", Arg: []byte("Example.Listener.onRoute")},
		{Name: "phase", Type: "string", Arg: []byte("both")},
	}
	if _, err := client.CallServerMethod("AddRouteListener", listenerArgs); err != nil {
		log.Printf("AddRouteListener failed: %v", err)
	} else {
		log.Println("Route listener for /welcome registered (phase: both)")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-c
	log.Println("Shutting down...")

	// 删除所有钩子
	hooks := []string{"/welcome", "/api/users/:id", "/static/*filepath"}
	for _, hook := range hooks {
		args = []*grpcloader.Arg{
			{Name: "class", Type: "string", Arg: []byte("onRequest")},
			{Name: "name", Type: "string", Arg: []byte(hook)},
		}
		if _, err := client.CallServerMethod("DeleteHook", args); err != nil {
			log.Printf("DeleteHook %s failed: %v", hook, err)
		} else {
			log.Printf("Hook %s deleted", hook)
		}
	}
	client.Unload()

}

func handleCommandStream(stream grpcloader.PluginService_NewCommandStreamClient) {
	for {
		cmd, err := stream.Recv()
		if err != nil {
			log.Printf("Command stream error: %v", err)
			return
		}

		var args []*grpcloader.Arg
		var handlerErr error

		switch cmd.Command {
		case "Example.Callback.onRequest":
			args, handlerErr = onRequest(cmd.Args)
		case "Example.Callback.onUserRequest":
			args, handlerErr = onUserRequest(cmd.Args)
		case "Example.Callback.onStaticRequest":
			args, handlerErr = onStaticRequest(cmd.Args)
		case "Example.Listener.onRoute":
			args, handlerErr = onRouteEvent(cmd.Args)
		case "heartbeat":
			args = []*grpcloader.Arg{
				{Name: "status", Type: "int", Arg: []byte("200")},
				{Name: "timestamp", Type: "int64", Arg: []byte(fmt.Sprintf("%d", time.Now().Unix()))},
			}
		default:
			log.Printf("Unknown command: %s", cmd.Command)
			args = []*grpcloader.Arg{
				{Name: "status", Type: "int", Arg: []byte("400")},
				{Name: "message", Type: "string", Arg: []byte("Unknown command: " + cmd.Command)},
			}
		}

		if handlerErr != nil {
			log.Printf("Failed to process %s: %v", cmd.Command, handlerErr)
			continue
		}

		if err := stream.Send(&grpcloader.Command{
			CommandId: cmd.CommandId,
			Command:   "return",
			Args:      args,
		}); err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}
}

// RequestInfo 请求信息结构体
type RequestInfo struct {
	Path    string              `json:"path"`
	Method  string              `json:"method"`
	Headers map[string][]string `json:"headers"`
	IP      string              `json:"ip"`
	TraceID string              `json:"traceID"`
	Params  map[string]string   `json:"params"` // 路由参数
	Body    []byte              `json:"body"`

	// 响应相关（响应阶段监听时提供）
	StatusCode      int                 `json:"statusCode"`
	ResponseHeaders map[string][]string `json:"responseHeaders"`
	ResponseBody    []byte              `json:"responseBody"`
	Phase           string              `json:"phase"`
}

// parseRequestArgs 解析请求参数
func parseRequestArgs(args []*grpcloader.Arg) *RequestInfo {
	info := &RequestInfo{
		Params: make(map[string]string),
	}
	for _, arg := range args {
		switch arg.Name {
		case "path":
			info.Path = string(arg.Arg)
		case "method":
			info.Method = string(arg.Arg)
		case "headers":
			json.Unmarshal(arg.Arg, &info.Headers)
		case "ip":
			info.IP = string(arg.Arg)
		case "traceID":
			info.TraceID = string(arg.Arg)
		case "params":
			json.Unmarshal(arg.Arg, &info.Params)
		case "body":
			info.Body = append([]byte(nil), arg.Arg...)
		case "statusCode":
			if code, err := strconv.Atoi(string(arg.Arg)); err == nil {
				info.StatusCode = code
			}
		case "responseHeaders":
			json.Unmarshal(arg.Arg, &info.ResponseHeaders)
		case "responseBody":
			info.ResponseBody = append([]byte(nil), arg.Arg...)
		case "phase":
			info.Phase = string(arg.Arg)
		}
	}
	return info
}

// onRequest 处理精确匹配路由 /welcome
func onRequest(args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)
	fmt.Printf("[onRequest] path=%s, method=%s, ip=%s\n", info.Path, info.Method, info.IP)

	return []*grpcloader.Arg{
		{Name: "statusCode", Type: "int", Arg: []byte("200")},
		{Name: "header", Type: "json", Arg: []byte(`{"Content-Type": "text/html", "X-Powered-By": "LiteBlog-Plugin"}`)},
		{Name: "body", Type: "[]byte", Arg: []byte("<h1>Welcome to LiteBlog!</h1><p>This is an exact match route.</p>")},
	}, nil
}

// onUserRequest 处理参数化路由 /api/users/:id
func onUserRequest(args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)

	// 从 params 中获取路由参数
	userID := info.Params["id"]
	fmt.Printf("[onUserRequest] path=%s, method=%s, userID=%s\n", info.Path, info.Method, userID)

	// 构建 JSON 响应
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":       userID,
			"name":     "User " + userID,
			"email":    "user" + userID + "@example.com",
			"path":     info.Path,
			"method":   info.Method,
			"clientIP": info.IP,
		},
	}
	responseBytes, _ := json.Marshal(response)

	return []*grpcloader.Arg{
		{Name: "statusCode", Type: "int", Arg: []byte("200")},
		{Name: "header", Type: "json", Arg: []byte(`{"Content-Type": "application/json", "X-Powered-By": "LiteBlog-Plugin"}`)},
		{Name: "body", Type: "[]byte", Arg: responseBytes},
	}, nil
}

// onStaticRequest 处理通配符路由 /static/*filepath
func onStaticRequest(args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)

	// 从 params 中获取通配符参数
	filepath := info.Params["filepath"]
	fmt.Printf("[onStaticRequest] path=%s, method=%s, filepath=%s\n", info.Path, info.Method, filepath)

	// 模拟静态文件响应
	body := fmt.Sprintf(`<h1>Static File Server</h1>
<p>Requested file: <code>%s</code></p>
<p>Full path: <code>%s</code></p>
<p>This demonstrates wildcard route matching with *filepath</p>`, filepath, info.Path)

	return []*grpcloader.Arg{
		{Name: "statusCode", Type: "int", Arg: []byte("200")},
		{Name: "header", Type: "json", Arg: []byte(`{"Content-Type": "text/html", "X-Powered-By": "LiteBlog-Plugin"}`)},
		{Name: "body", Type: "[]byte", Arg: []byte(body)},
	}, nil
}

// onRouteEvent 路由监听示例（请求/响应均会回调）
func onRouteEvent(args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	info := parseRequestArgs(args)
	phase := info.Phase
	if phase == "" {
		phase = "response"
	}

	log.Printf("[onRouteEvent][%s] %s %s trace=%s status=%d", phase, info.Method, info.Path, info.TraceID, info.StatusCode)

	if len(info.Body) > 0 {
		log.Printf("[onRouteEvent] request body (%d bytes): %.200s", len(info.Body), string(info.Body))
	}

	if phase != "request" {
		log.Printf("[onRouteEvent] response headers: %+v", info.ResponseHeaders)
		if len(info.ResponseBody) > 0 {
			log.Printf("[onRouteEvent] response body (%d bytes): %.200s", len(info.ResponseBody), string(info.ResponseBody))
		}
	}

	return []*grpcloader.Arg{
		{Name: "ack", Type: "bool", Arg: []byte("true")},
	}, nil
}
