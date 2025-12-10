package main_test

import (
	"context"
	"fmt"
	"testing"

	grpcloader "LiteBlog/utils/plugins/gRPCLoader"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestGRPCPlugin(t *testing.T) {
	t.Log("Testing GRPC Plugin")
	conn, err := grpc.NewClient("localhost:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Error(err)
	}
	client := grpcloader.NewPluginServiceClient(conn)
	ctx := context.Background()
	// init
	i, err := client.Initialize(ctx, &grpcloader.Empty{})
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Init: %s,id: %s\n", i.Version, i.Id)
	md := metadata.Pairs("id", i.Id)
	ctx = metadata.NewOutgoingContext(ctx, md)
	// create bidistream
	bidstream, err := client.NewCommandStream(ctx)
	if err != nil {
		t.Error(err)
	}
	go func() {
		for {
			cmd, err := bidstream.Recv()
			if err != nil {
				fmt.Printf("Recv error: %s\n", err)
				break
			}
			fmt.Printf("Recv: %s\n", cmd.Command)
			switch cmd.Command {
			case "test2":
				fmt.Printf("Test2\n")
			}
			bidstream.Send(&grpcloader.Command{
				Command:   "return",
				CommandId: cmd.CommandId,
				Args: []*grpcloader.Arg{
					{
						Type: "string",
						Name: "result",
						Arg:  []byte("success"),
					},
				},
			})
		}
	}()
	// get and register methods
	m, err := client.RegisterPluginMethods(ctx, &grpcloader.RegisterMethodsRequest{
		Methods: []*grpcloader.MethodDef{
			{
				Name: "test2",
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
	for _, method := range m.Methods {
		fmt.Printf("Method: %s, Args: %s\n", method.Name, method.ArgsNames)
		// try to call
		client.CallServerMethod(ctx, &grpcloader.CallMethod{
			Method: method.Name,
			Args: []*grpcloader.Arg{
				{
					Type: "string",
					Arg:  []byte("hello"),
					Name: "arg",
				},
			},
		})
	}
	if err != nil {
		t.Error(err)
	}
	select {}
}
