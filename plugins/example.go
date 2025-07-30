package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcloader "LiteBlog/utils/plugins/gRPCLoader"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type PluginClient struct {
	conn   *grpc.ClientConn
	client grpcloader.PluginServiceClient
	id     string
}

func NewPluginClient(serverAddr string) (*PluginClient, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	client := grpcloader.NewPluginServiceClient(conn)
	return &PluginClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *PluginClient) Initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

func (c *PluginClient) RegisterMethods(methods []*grpcloader.MethodDef) error {
	ctx, cancel := context.WithTimeout(c.withIDContext(context.Background()), 5*time.Second)
	defer cancel()

	_, err := c.client.GetRegisteredMethods(ctx, &grpcloader.RegisterMethods{
		Methods: methods,
	})
	return err
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
		log.Fatal("Usage: client <server-address>")
	}
	serverAddr := os.Args[1]

	// 创建客户端
	client, err := NewPluginClient(serverAddr)
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
		{
			Name: "Example.Callback.onRequest",
		},
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

	// 调用服务器方法示例
	args := []*grpcloader.Arg{
		{Name: "class", Type: "string", Arg: []byte("onRequest")},                     // 类名
		{Name: "callback", Type: "string", Arg: []byte("Example.Callback.onRequest")}, // 方法名
		{Name: "name", Type: "string", Arg: []byte("/welcome")},                       // 请求路径
	}
	results, err := client.CallServerMethod("AddHook", args) // AddHook - 添加一个钩子
	if err != nil {
		log.Printf("Method call failed: %v", err)
	} else {
		log.Printf("Method call results: %+v", results)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-c
	log.Println("Shutting down...")
	args = []*grpcloader.Arg{
		{
			Name: "class",
			Type: "string",
			Arg:  []byte("onRequest"),
		},
		{
			Name: "callback",
			Type: "string",
			Arg:  []byte("Example.Callback.onRequest"),
		},
		{
			Name: "name",
			Type: "string",
			Arg:  []byte("/welcome"),
		},
	}
	if _, err := client.CallServerMethod("DeleteHook", args); err != nil { // 卸载钩子
		log.Printf("Method call failed: %v", err)
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

		// log.Printf("Received command: %s", cmd.Command)
		switch cmd.Command {
		case "Example.Callback.onRequest":
			args, err := onRequest(cmd.Args)
			if err != nil {
				log.Printf("Failed to process request: %v", err)
				continue
			}
			if err := stream.Send(&grpcloader.Command{ // send return args
				CommandId: cmd.CommandId,
				Command:   "return",
				Args:      args,
			}); err != nil {
				log.Printf("Failed to send response: %v", err)
			}
		case "heartbeat":
			// 处理心跳包
			if err := stream.Send(&grpcloader.Command{
				CommandId: cmd.CommandId,
				Command:   "return",
				Args: []*grpcloader.Arg{{Name: "status", Type: "int", Arg: []byte("200")},
					{Name: "timestamp", Type: "int64", Arg: []byte(fmt.Sprintf("%d", time.Now().Unix()))}},
			}); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
			}
		default: // unknown command
			log.Printf("Unknown command: %s", cmd.Command)
			stream.Send(&grpcloader.Command{
				CommandId: cmd.CommandId,
				Command:   "return",
				Args: []*grpcloader.Arg{{Name: "status", Type: "int", Arg: []byte("400")},
					{Name: "message", Type: "string", Arg: []byte("Unknown command: " + cmd.Command)}},
			})
		}
	}
}

func onRequest(args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	var path string
	var method string
	var headers map[string][]string
	var ip string
	var traceID string
	for _, arg := range args {
		switch arg.Name {
		case "path":
			path = string(arg.Arg)
		case "method":
			method = string(arg.Arg)
		case "headers":
			json.Unmarshal(arg.Arg, &headers)
		case "ip":
			ip = string(arg.Arg)
		case "traceID":
			traceID = string(arg.Arg)
		}
	}
	fmt.Printf("Received request: path=%s, method=%s, headers=%+v, ip=%s, traceID=%s\n", path, method, headers, ip, traceID)
	// 处理请求
	ret := []*grpcloader.Arg{
		{
			Name: "statusCode",
			Type: "int",
			Arg:  []byte("200"),
		},
		{
			Name: "header",
			Type: "json-map[string][]string",
			Arg:  []byte(`{"Content-Type": ["text/plain"], "X-Powered-By": ["LiteBlog-PluginExample"]}`),
		},
		{
			Name: "body",
			Type: "[]byte",
			Arg:  []byte("Hello, LiteBlog plugin world!"),
		},
	}
	return ret, nil

}
