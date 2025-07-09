package grpcloader

import (
	context "context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

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
	availableMethods         sync.Map // map[string]*MethodDef // key is method name, value is a struct with MethodDef and ID
	pluginMethods            sync.Map // map[string]*struct {*MethodDef,ID string} // key is method name, value is a struct with MethodDef and ID
	status                   int      // 0: not started, 1: started, 2: stopped
	ctx                      context.Context
	cancle                   context.CancelFunc
	LoadedIds                []string
	idChangeLocker           sync.Mutex
	id2bidStream             sync.Map                                              // map[string]grpc.BidiStreamingServer[Command, Command]
	id2methods               sync.Map                                              // map[string][]string // key is plugin id, value is a list of method names
	commandQueueChannel      sync.Map                                              // map[string]chan []*Arg // command queue for each bidstream, key is command id
	Id2QueueChannel          sync.Map                                              // map[string]map[string]chan []*Arg // command queue for each command id, key is plugin id, map[commandId]chan []*Arg
	methodHandler            func(string, []*Arg) ([]*Arg, error)                  // server method handler
	unregisterMethodsHandler func([]string)                                        // unregister methods handler, used to call unregister methods when plugin is unloaded
	pluginMethodhandler      func(map[string]func(string, []*Arg) ([]*Arg, error)) // plugin method handler
}

// gRPC function
func (gpl *GRPCPluginLoader) Initialize(ctx context.Context, e *Empty) (*InitializeResponse, error) {
	gpl.idChangeLocker.Lock()
	defer gpl.idChangeLocker.Unlock()
	id := generateVerifyCode(8)                // generate plugin id
	gpl.LoadedIds = append(gpl.LoadedIds, id)  // add plugin id to loaded ids
	gpl.Id2QueueChannel.Store(id, &sync.Map{}) // add command queue channel to Id2QueueChannel
	resp := &InitializeResponse{Id: id, Version: gPRC_Plugin_Interface_Version}

	return resp, nil
}

// gRPC function, Unload plugin
func (gpl *GRPCPluginLoader) UnLoad(ctx context.Context, v *Verify) (*Error, error) {
	gpl.idChangeLocker.Lock()
	defer gpl.idChangeLocker.Unlock()
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
			methodsA, ok := gpl.id2methods.Load(loadedId)
			if !ok {
				return nil, ErrIdNotFound
			}
			methods := methodsA.([]string)
			for _, method := range methods {
				gpl.pluginMethods.Delete(method) // delete plugin method
			}
			gpl.unregisterMethodsHandler(methods) // broadcast to loader, call unregister methods handler
			// delete bidstream
			_, ok = gpl.id2bidStream.Load(loadedId) // get bidstream
			if ok {
				gpl.id2bidStream.Delete(loadedId) // delete bidstream
			}
			// delete command queue channel
			if qmapA, ok := gpl.Id2QueueChannel.Load(id); ok {
				qmap, ok := qmapA.(*sync.Map) // get command queue channel map, map[commandId]chan []*Arg
				if !ok {
					return nil, ErrIdNotFound
				}
				qmap.Range(func(key, value any) bool {
					commId := key.(string)
					gpl.commandQueueChannel.Delete(commId)
					ch := value.(chan []*Arg)
					if ch != nil {
						close(ch)
					}
					qmap.Delete(commId)
					return true
				})
				gpl.Id2QueueChannel.Delete(id) // delete command queue channel
			}

			gpl.LoadedIds = append(gpl.LoadedIds[:i], gpl.LoadedIds[i+1:]...) // remove plugin id from loaded ids
			// remove plugin methods
			gpl.id2methods.Delete(loadedId)
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
	rm := make([]*MethodDef, 0, LengthOfMap(&gpl.availableMethods))
	gpl.availableMethods.Range(func(key, value any) bool {
		rm = append(rm, value.(*MethodDef))
		return true
	})
	pluginMethodMap := make(map[string]func(string, []*Arg) ([]*Arg, error))
	for _, method := range pluginMethods.Methods { // record plugin methods
		// check if method has been registered
		if _, ok := gpl.pluginMethods.Load(method.Name); ok {
			return nil, ErrMethodRepeated
		}
		listA, _ := gpl.id2methods.LoadOrStore(id, []string{})
		list := listA.([]string)
		gpl.id2methods.Store(id, append(list, method.Name)) // record method names
		gpl.pluginMethods.Store(method.Name, &struct {
			*MethodDef
			ID string
		}{method, id})
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
	if _, ok := gpl.id2bidStream.Load(id); ok {
		return ErrIdExists
	}
	gpl.id2bidStream.Store(id, bidstream)
	// create heartbeat goroutine
	go gpl.startHeartbeat(bidstream)
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
func (gpl *GRPCPluginLoader) startHeartbeat(stream grpc.ServerStream) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			stream.SendMsg(&Command{Command: "heartbeat"})
		case <-stream.Context().Done():
			return
		}
	}
}

// original code
func (gpl *GRPCPluginLoader) CallClientMethod(method string, args []*Arg) ([]*Arg, error) {
	// get bidstream
	idA, ok := gpl.pluginMethods.Load(method)
	if !ok {
		return nil, ErrMethodNotFound
	}
	id := idA.(*struct {
		*MethodDef
		ID string
	}).ID
	if id == "" {
		return nil, ErrMethodNotFound
	}
	bidstreamA, ok := gpl.id2bidStream.Load(id)
	if !ok {
		return nil, ErrBidStreamNotFound
	}
	bidstream := bidstreamA.(grpc.BidiStreamingServer[Command, Command])
	ctx := bidstream.Context()
	commId := generateVerifyCode(8)
	newchan := make(chan []*Arg, 1)
	gpl.commandQueueChannel.Store(commId, newchan) // add command queue channel to commandQueueChannel
	v, ok := gpl.Id2QueueChannel.Load(id)          // map[string]chan []*Arg
	if !ok {
		return nil, ErrIdNotFound
	}
	mchan := v.(*sync.Map)
	mchan.Store(commId, newchan) // add command queue channel to Id2QueueChannel
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
		gpl.commandQueueChannel.Delete(commId) // delete command queue channel
		mchan.Delete(commId)                   // delete command queue channel
		return retargs, nil
	case <-ctx.Done(): // connection closed
		gpl.commandQueueChannel.Delete(commId) // delete command queue channel
		mchan.Delete(commId)                   // delete command queue channel
		return nil, nil
	}

}

// original code
func (gpl *GRPCPluginLoader) Shutdown() error {
	switch gpl.status {
	case 0:
		return nil
	case 1:
		gpl.commandQueueChannel.Range(func(key, value interface{}) bool {
			ch := value.(chan []*Arg)
			if ch != nil {
				close(ch)
			}
			gpl.commandQueueChannel.Delete(key)
			return true
		})
		gpl.cancle()
		gpl.status = 2
		gpl.availableMethods.Clear()
		gpl.Id2QueueChannel.Clear()
		gpl.pluginMethods.Clear()
		gpl.id2bidStream.Clear()
		gpl.id2methods.Clear()
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
	gpl.status = 1
	return nil
}

// original code
func (gpl *GRPCPluginLoader) SetMethods(methods map[string]*MethodDef) {
	for _, method := range methods {
		gpl.availableMethods.Store(method.Name, method)
	}
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
	ctx := stream.Context()
	id, err := getIdFromContext(ctx)
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done(): // connection closed
			gpl.UnLoad(ctx, &Verify{Id: id})
			return
		default:
			cmd := &Command{}
			if err := stream.RecvMsg(cmd); err != nil {
				// call unload method
				gpl.UnLoad(ctx, &Verify{Id: id})
				break
			}
			// check commnd id
			chanbackA, ok := gpl.commandQueueChannel.Load(cmd.CommandId)
			if !ok {
				continue
			}
			chanback := chanbackA.(chan []*Arg)
			switch cmd.Command {
			case "return":
				chanback <- cmd.Args
			default:

			}
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
	select {
	case <-ctx.Done(): // connection closed
		return "", ctx.Err()
	default:
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

}

func LengthOfMap(m *sync.Map) int {
	length := 0
	m.Range(func(key, value any) bool {
		length++
		return true
	})
	return length
}
