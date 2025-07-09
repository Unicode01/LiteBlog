package main

import (
	"context"
	"fmt"
	"log"
	"os"
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
	// 建立gRPC连接
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

	// 1. 创建客户端
	client, err := NewPluginClient(serverAddr)
	if err != nil {
		log.Fatalf("Client creation failed: %v", err)
	}
	defer client.Close()

	// 2. 初始化插件
	if err := client.Initialize(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	// 3. 注册方法
	methods := []*grpcloader.MethodDef{
		{
			Name: "exampleMethod",
			ArgsNames: []*grpcloader.Arg{
				{Name: "param1", Type: "string"},
				{Name: "param2", Type: "int"},
			},
		},
	}
	if err := client.RegisterMethods(methods); err != nil {
		log.Printf("Method registration failed: %v", err)
	} else {
		log.Println("Methods registered successfully")
	}

	// 4. 调用服务器方法示例
	args := []*grpcloader.Arg{
		{Name: "input", Type: "string", Arg: []byte("Hello from client")},
	}
	results, err := client.CallServerMethod("Load", args)
	if err != nil {
		log.Printf("Method call failed: %v", err)
	} else {
		log.Printf("Method call results: %+v", results)
	}

	// 5. 创建命令流示例
	stream, err := client.NewCommandStream()
	if err != nil {
		log.Printf("Command stream creation failed: %v", err)
	} else {
		go handleCommandStream(stream)
	}

	// 6. 卸载插件
	log.Println("Unloading plugin...")
	client.CallServerMethod("Unload", nil)
	if err := client.Unload(); err != nil {
		log.Printf("Unload failed: %v", err)
	} else {
		log.Println("Unloaded successfully")
	}
}

func handleCommandStream(stream grpcloader.PluginService_NewCommandStreamClient) {
	for {
		cmd, err := stream.Recv()
		if err != nil {
			log.Printf("Command stream error: %v", err)
			return
		}

		log.Printf("Received command: %s", cmd.Command)

		// 处理命令并返回响应
		if err := stream.Send(&grpcloader.Command{
			CommandId: cmd.CommandId,
			Command:   "response",
			Args: []*grpcloader.Arg{
				{Name: "result", Type: "string", Arg: []byte("Processed: " + cmd.Command)},
			},
		}); err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}
}
