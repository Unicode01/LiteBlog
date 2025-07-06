package grpcloader

import (
	context "context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

var (
	gPRC_Plugin_Interface_Version = "0.0.1"

	ErrPermissionDenied  = status.Error(codes.PermissionDenied, "permission denied")
	ErrIdExists          = status.Error(codes.AlreadyExists, "id already exists")
	ErrMethodNotFound    = status.Error(codes.NotFound, "method not found")
	ErrBidStreamNotFound = status.Error(codes.NotFound, "bidstream not found")
	ErrUnknown           = status.Error(codes.Unknown, "unknown error")
	ErrMissingMetadata   = status.Error(codes.InvalidArgument, "missing metadata")
	ErrIdNotFound        = status.Error(codes.NotFound, "id not found")
	ErrMethodRepeated    = status.Error(codes.AlreadyExists, "method repeated")
)

type GRPCPluginLoader struct {
	UnimplementedPluginServiceServer
	availableMethods map[string]*MethodDef
	pluginMethods    map[string]*struct {
		*MethodDef
		ID string
	}
	status                   int // 0: not started, 1: started, 2: stopped
	ctx                      context.Context
	cancle                   context.CancelFunc
	LoadedIds                []string
	id2bidStream             map[string]grpc.BidiStreamingServer[Command, Command]
	id2methods               map[string][]string                                   // key is plugin id, value is a list of method names
	commandQueueChannel      map[string]chan []*Arg                                // command queue for each bidstream, key is command id
	methodHandler            func(string, []*Arg) ([]*Arg, error)                  // server method handler
	unregisterMethodsHandler func([]string)                                        // unregister methods handler, used to call unregister methods when plugin is unloaded
	pluginMethodhandler      func(map[string]func(string, []*Arg) ([]*Arg, error)) // plugin method handler
}

// gRPC function
func (gpl *GRPCPluginLoader) Initialize(ctx context.Context, e *Empty) (*InitializeResponse, error) {
	fmt.Printf("Initialize called\n")
	id := generateVerifyCode(4)               // generate plugin id
	gpl.LoadedIds = append(gpl.LoadedIds, id) // add plugin id to loaded ids
	resp := &InitializeResponse{Id: id, Version: gPRC_Plugin_Interface_Version}

	return resp, nil
}

// gRPC function, Unload plugin
func (gpl *GRPCPluginLoader) UnLoad(ctx context.Context, v *Verify) (*Error, error) {
	sid := v.Id
	id, err := getIdFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if id != sid {
		return nil, ErrPermissionDenied
	}
	for i, loadedId := range gpl.LoadedIds {
		if loadedId == id {
			// unregister plugin methods
			methods := gpl.id2methods[loadedId]
			for _, method := range methods {
				delete(gpl.pluginMethods, method) // delete plugin method
			}
			gpl.unregisterMethodsHandler(methods) // broadcast to loader, call unregister methods handler
			// delete bidstream
			_, ok := gpl.id2bidStream[loadedId] // get bidstream
			if ok {
				delete(gpl.id2bidStream, loadedId) // delete bidstream
			}
			// delete command queue channel
			if ch, ok := gpl.commandQueueChannel[id]; ok {
				close(ch)
				delete(gpl.commandQueueChannel, id)
			}
			gpl.LoadedIds = append(gpl.LoadedIds[:i], gpl.LoadedIds[i+1:]...) // remove plugin id from loaded ids
			// remove plugin methods
			delete(gpl.id2methods, loadedId)
			break
		}
	}
	return nil, nil
}

// gRPC function
func (gpl *GRPCPluginLoader) GetRegisteredMethods(ctx context.Context, pluginMethods *RegisterMethods) (*RegisterMethods, error) {
	id, err := getIdFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rm := make([]*MethodDef, 0, len(gpl.availableMethods))
	for _, method := range gpl.availableMethods { // provide available methods
		rm = append(rm, method)
	}
	pluginMethodMap := make(map[string]func(string, []*Arg) ([]*Arg, error))
	for _, method := range pluginMethods.Methods { // record plugin methods
		// check if method has been registered
		if _, ok := gpl.pluginMethods[method.Name]; ok {
			return nil, ErrMethodRepeated
		}
		gpl.id2methods[id] = append(gpl.id2methods[id], method.Name) // record method names
		gpl.pluginMethods[method.Name] = &struct {
			*MethodDef
			ID string
		}{method, id}
		pluginMethodMap[method.Name] = func(s string, a []*Arg) ([]*Arg, error) {
			args := []*Arg{}
			for _, arg := range a {
				args = append(args, &Arg{
					Type: arg.Type,
					Arg:  arg.Arg,
					Name: arg.Name,
				})
			}
			return gpl.CallClientMethod(method.Name, args)
		}
	}
	if gpl.pluginMethodhandler != nil {
		gpl.pluginMethodhandler(pluginMethodMap)
	}

	return &RegisterMethods{Methods: rm}, nil
}

// gRPC function
func (gpl *GRPCPluginLoader) NewCommandStream(bidstream grpc.BidiStreamingServer[Command, Command]) error {
	id, err := getIdFromContext(bidstream.Context())
	if err != nil {
		return err
	}
	if _, ok := gpl.id2bidStream[id]; ok {
		return ErrIdExists
	}
	gpl.id2bidStream[id] = bidstream
	// create a new goroutine to handle the bidstream
	gpl.clientCommendListener(bidstream) // isRecv = false, send to client
	return nil
}

// gRPC function
func (gpl *GRPCPluginLoader) CallServerMethod(ctx context.Context, cm *CallMethod) (*Args, error) {
	ArgsRt, err := gpl.methodHandler(cm.Method, cm.Args)
	return &Args{Args: ArgsRt}, err
}

// original code, unregister plugin methods,
// used in plugin unload
func (gpl *GRPCPluginLoader) SetUnregisterMethodsHandler(f func([]string)) {
	gpl.unregisterMethodsHandler = f
}

// original code
func (gpl *GRPCPluginLoader) CallClientMethod(method string, args []*Arg) ([]*Arg, error) {
	// get bidstream
	id := gpl.pluginMethods[method].ID
	if id == "" {
		return nil, ErrMethodNotFound
	}
	bidstream, ok := gpl.id2bidStream[id]
	ctx := bidstream.Context()
	if !ok {
		return nil, ErrBidStreamNotFound
	}
	commId := generateVerifyCode(4)
	newchan := make(chan []*Arg, 1)
	gpl.commandQueueChannel[commId] = newchan
	// send command
	cmd := &Command{
		Command:   method,
		Args:      args,
		CommandId: commId,
	}
	if err := bidstream.Send(cmd); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// wait for response
	select {
	case retargs := <-newchan:
		delete(gpl.commandQueueChannel, commId)
		return retargs, nil
	case <-ctx.Done(): // connection closed
		return nil, nil
	}

}

// original code
func (gpl *GRPCPluginLoader) Shutdown() error {
	switch gpl.status {
	case 0:
		return nil
	case 1:
		for id, ch := range gpl.commandQueueChannel {
			close(ch)
			delete(gpl.commandQueueChannel, id)
		}
		gpl.cancle()
		gpl.status = 2
		gpl.availableMethods = nil
		gpl.pluginMethods = nil
		gpl.id2bidStream = nil
		gpl.id2methods = nil
	case 2:
		return nil
	default:
		return ErrUnknown
	}
	return nil
}

// original code
func (gpl *GRPCPluginLoader) Init() error {
	gpl.ctx, gpl.cancle = context.WithCancel(context.Background())
	gpl.availableMethods = make(map[string]*MethodDef)
	gpl.pluginMethods = make(map[string]*struct {
		*MethodDef
		ID string
	})
	gpl.id2bidStream = make(map[string]grpc.BidiStreamingServer[Command, Command])
	gpl.id2methods = make(map[string][]string)
	gpl.commandQueueChannel = make(map[string]chan []*Arg)
	gpl.status = 1
	return nil
}

// original code
func (gpl *GRPCPluginLoader) SetMethods(methods map[string]*MethodDef) {
	gpl.availableMethods = methods
}

// original code
func (gpl *GRPCPluginLoader) SetMethodHandler(handler func(string, []*Arg) ([]*Arg, error)) {
	gpl.methodHandler = handler
}

// original code
func (gpl *GRPCPluginLoader) SetPluginMethodHandler(handler func(map[string]func(string, []*Arg) ([]*Arg, error))) {
	gpl.pluginMethodhandler = handler
}

func (gpl *GRPCPluginLoader) clientCommendListener(stream grpc.ServerStream) {
	id, err := getIdFromContext(stream.Context())
	if err != nil {
		return
	}
	for {
		cmd := &Command{}
		if err := stream.RecvMsg(cmd); err != nil {
			// call unload method
			gpl.UnLoad(stream.Context(), &Verify{Id: id})
			break
		}
		// check commnd id
		chanback, ok := gpl.commandQueueChannel[cmd.CommandId]
		if !ok {
			continue
		}
		switch cmd.Command {
		case "return":
			chanback <- cmd.Args
		default:
		}

	}
}

// tool function
func generateVerifyCode(lenth ...int) string {
	if len(lenth) == 0 {
		lenth = []int{4}
	}
	randBytes := make([]byte, lenth[0])
	rand.Read(randBytes)
	vc := hex.EncodeToString(randBytes)
	return vc
}

// tool function
func getIdFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrMissingMetadata
	}
	idValues := md.Get("id")
	if len(idValues) == 0 {
		return "", ErrIdNotFound
	}
	id := idValues[0]
	return id, nil
}
