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
	ErrInvalidAccessKey  = status.Error(codes.Unauthenticated, "invalid access key")
	ErrIdExists          = status.Error(codes.AlreadyExists, "id already exists")
	ErrMethodNotFound    = status.Error(codes.NotFound, "method not found")
	ErrBidStreamNotFound = status.Error(codes.NotFound, "bidstream not found")
	ErrUnknown           = status.Error(codes.Unknown, "unknown error")
	ErrMissingMetadata   = status.Error(codes.InvalidArgument, "missing metadata")
	ErrIdNotFound        = status.Error(codes.NotFound, "id not found")
	ErrMethodRepeated    = status.Error(codes.AlreadyExists, "method repeated")
	ErrMissingAccessKey  = status.Error(codes.Unauthenticated, "missing access key")
)

type GRPCPluginLoader struct {
	UnimplementedPluginServiceServer
	availableMethods         sync.Map // map[string]*MethodDef // key is method name, value is a struct with MethodDef and ID
	pluginMethods            sync.Map // map[string]*struct {*MethodDef,ID string} // key is method name, value is a struct with MethodDef and ID
	status                   int      // 0: not started, 1: started, 2: stopped
	ctx                      context.Context
	cancle                   context.CancelFunc
	loadedIds                sync.Map                                              // map[string]bool // 使用 sync.Map 替代切片，保证并发安全
	idChangeLocker           sync.Mutex                                            // 保留用于初始化时的顺序控制
	id2bidStream             sync.Map                                              // map[string]grpc.BidiStreamingServer[Command, Command]
	id2methods               sync.Map                                              // map[string][]string // key is plugin id, value is a list of method names
	commandQueueChannel      sync.Map                                              // map[string]chan []*Arg // command queue for each bidstream, key is command id
	Id2QueueChannel          sync.Map                                              // map[string]map[string]chan []*Arg // command queue for each command id, key is plugin id, map[commandId]chan []*Arg
	methodHandler            func(string, []*Arg) ([]*Arg, error)                  // server method handler
	unregisterMethodsHandler func([]string)                                        // unregister methods handler, used to call unregister methods when plugin is unloaded
	pluginMethodhandler      func(map[string]func(string, []*Arg) ([]*Arg, error)) // plugin method handler
	commandTimeout           time.Duration                                         // 命令超时时间
	accessKey                string                                                // 访问密钥，为空则不验证
}

// gRPC function
func (gpl *GRPCPluginLoader) Initialize(ctx context.Context, e *Empty) (*InitializeResponse, error) {
	// 验证 access key（如果配置了）
	if gpl.accessKey != "" {
		if err := gpl.verifyAccessKey(ctx); err != nil {
			return nil, err
		}
	}

	gpl.idChangeLocker.Lock()
	defer gpl.idChangeLocker.Unlock()
	id := generateVerifyCode(8)                // generate plugin id
	gpl.loadedIds.Store(id, true)              // add plugin id to loaded ids (并发安全)
	gpl.Id2QueueChannel.Store(id, &sync.Map{}) // add command queue channel to Id2QueueChannel
	resp := &InitializeResponse{Id: id, Version: gPRC_Plugin_Interface_Version}

	return resp, nil
}

// verifyAccessKey 验证访问密钥
func (gpl *GRPCPluginLoader) verifyAccessKey(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ErrMissingAccessKey
	}
	keys := md.Get("access-key")
	if len(keys) == 0 {
		return ErrMissingAccessKey
	}
	if keys[0] != gpl.accessKey {
		return ErrInvalidAccessKey
	}
	return nil
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

	// 检查插件是否存在
	if _, ok := gpl.loadedIds.Load(id); !ok {
		return nil, ErrIdNotFound
	}

	// unregister plugin methods
	if methodsA, ok := gpl.id2methods.Load(id); ok {
		methods := methodsA.([]string)
		for _, method := range methods {
			gpl.pluginMethods.Delete(method) // delete plugin method
		}
		if gpl.unregisterMethodsHandler != nil {
			gpl.unregisterMethodsHandler(methods) // broadcast to loader, call unregister methods handler
		}
	}

	// delete bidstream
	gpl.id2bidStream.Delete(id)

	// delete command queue channel
	if qmapA, ok := gpl.Id2QueueChannel.Load(id); ok {
		if qmap, ok := qmapA.(*sync.Map); ok {
			qmap.Range(func(key, value any) bool {
				commId := key.(string)
				gpl.commandQueueChannel.Delete(commId)
				if ch, ok := value.(chan []*Arg); ok && ch != nil {
					close(ch)
				}
				return true
			})
		}
		gpl.Id2QueueChannel.Delete(id)
	}

	// remove from loaded ids (并发安全)
	gpl.loadedIds.Delete(id)
	// remove plugin methods mapping
	gpl.id2methods.Delete(id)

	return nil, nil
}

// gRPC function - 插件注册其方法，服务器返回可用的公共方法列表
func (gpl *GRPCPluginLoader) RegisterPluginMethods(ctx context.Context, req *RegisterMethodsRequest) (*AvailableMethodsResponse, error) {
	id, err := getIdFromContext(ctx)
	if err != nil {
		return nil, err
	}
	// 收集服务器可用的公共方法
	availableMethods := make([]*MethodDef, 0, LengthOfMap(&gpl.availableMethods))
	gpl.availableMethods.Range(func(key, value any) bool {
		availableMethods = append(availableMethods, value.(*MethodDef))
		return true
	})
	// 注册插件方法
	pluginMethodMap := make(map[string]func(string, []*Arg) ([]*Arg, error))
	for _, method := range req.Methods {
		// 检查方法是否已被注册
		if _, ok := gpl.pluginMethods.Load(method.Name); ok {
			return nil, ErrMethodRepeated
		}
		listA, _ := gpl.id2methods.LoadOrStore(id, []string{})
		list := listA.([]string)
		gpl.id2methods.Store(id, append(list, method.Name))
		gpl.pluginMethods.Store(method.Name, &struct {
			*MethodDef
			ID string
		}{method, id})
		// 捕获当前方法名
		currentMethodName := method.Name
		pluginMethodMap[currentMethodName] = func(s string, a []*Arg) ([]*Arg, error) {
			args := make([]*Arg, len(a))
			for i, arg := range a {
				args[i] = &Arg{
					Type: arg.Type,
					Arg:  arg.Arg,
					Name: arg.Name,
				}
			}
			return gpl.CallClientMethod(currentMethodName, args)
		}
	}
	if gpl.pluginMethodhandler != nil {
		gpl.pluginMethodhandler(pluginMethodMap)
	}

	return &AvailableMethodsResponse{Methods: availableMethods}, nil
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

	// cleanup helper
	cleanup := func() {
		gpl.commandQueueChannel.Delete(commId)
		mchan.Delete(commId)
	}

	// send command
	cmd := &Command{
		Command:   method,
		Args:      args,
		CommandId: commId,
	}
	if err := bidstream.Send(cmd); err != nil {
		cleanup()
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 获取超时时间，默认 30 秒
	timeout := gpl.commandTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// wait for response with timeout
	select {
	case retargs := <-newchan:
		cleanup()
		return retargs, nil
	case <-time.After(timeout): // 超时机制
		cleanup()
		return nil, status.Error(codes.DeadlineExceeded, "plugin method call timeout")
	case <-ctx.Done(): // connection closed
		cleanup()
		return nil, status.Error(codes.Canceled, "connection closed")
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
	gpl.commandTimeout = 30 * time.Second // 默认超时时间
	return nil
}

// SetCommandTimeout 设置命令超时时间
func (gpl *GRPCPluginLoader) SetCommandTimeout(timeout time.Duration) {
	gpl.commandTimeout = timeout
}

// SetAccessKey 设置访问密钥
func (gpl *GRPCPluginLoader) SetAccessKey(key string) {
	gpl.accessKey = key
}

// GetLoadedPluginIDs 获取已加载的插件ID列表
func (gpl *GRPCPluginLoader) GetLoadedPluginIDs() []string {
	ids := make([]string, 0)
	gpl.loadedIds.Range(func(key, value any) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
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
