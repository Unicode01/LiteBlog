// plugin:
// set up for plugins, including loading and initialization

package plugins

import (
	"context"
	"errors"
	"net"
	"sync"

	grpcloader "LiteBlog/utils/plugins/gRPCLoader"

	"google.golang.org/grpc"
)

var (
	ErrMethodNotFound = errors.New("method not found")
)

type PluginLoader struct {
}

func NewPluginLoader() *PluginLoader {
	pl := &PluginLoader{}
	return pl
}

type Loader struct {
	id string
}

type LoaderType interface {
	ID() string                                                                         // loader method, Get Loader ID
	SetID(id string)                                                                    // loader method, Set Loader ID
	Init() error                                                                        // loader method, initialize plugin loader
	Load() error                                                                        // plugin method, load plugin loader
	Unload() error                                                                      // plugin method, unload plugin loader
	UnloadPlugin(pluginID string) error                                                 // plugin method, call unload function of plugin
	RegisterMethods(methods map[string]func([]*Arg) ([]*Arg, error)) error              // plugin method, register server methods
	SetPluginMethodHandler(handler func(map[string]func([]*Arg) ([]*Arg, error))) error // plugin method, register plugin method handler
	SetUnregisterMethodsHandler(handler func([]string)) error                           // plugin method, register unregister methods handler
	CallPluginMethod(method string, args []*Arg) ([]*Arg, error)                        // plugin method, call plugin method
	UnregisterMethods(methods []string) error                                           // plugin method, unregister server methods
}

// create a new plugin loader instance,
// support load from gRPC
type LoaderTypeGRPC struct {
	Loader
	ListenerAddress  string // loader listener address e.g. 127.0.0.1:8080 [::]:8080 , better not use public ip address
	grpcServer       *grpc.Server
	grpcLoaderServer *grpcloader.GRPCPluginLoader
	ctx              context.Context
	cancle           context.CancelFunc
	publicMethods    sync.Map // map[string]func([]*Arg) ([]*Arg, error)
}

func (ltgrpc *LoaderTypeGRPC) Init() error {
	lis, err := net.Listen("tcp", ltgrpc.ListenerAddress)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	ltgrpc.grpcServer = s
	ltgrpc.grpcLoaderServer = &grpcloader.GRPCPluginLoader{}
	ltgrpc.grpcLoaderServer.Init()
	grpcloader.RegisterPluginServiceServer(s, ltgrpc.grpcLoaderServer)
	ltgrpc.grpcLoaderServer.SetMethodHandler(ltgrpc.CallMethod)
	ltgrpc.ctx, ltgrpc.cancle = context.WithCancel(context.Background())
	go func() {
		err = s.Serve(lis)
		if err != nil {
			ltgrpc.cancle()
		}
	}()
	return nil
}

func (ltgrpc *LoaderTypeGRPC) Load() error {
	return nil
}

func (ltgrpc *LoaderTypeGRPC) RegisterMethods(methods map[string]func([]*Arg) ([]*Arg, error)) error {
	for method, f := range methods {
		ltgrpc.publicMethods.Store(method, f)
	}
	ms := map[string]*grpcloader.MethodDef{}
	for method := range methods {
		m := &grpcloader.MethodDef{
			Name: method,
		}
		ms[method] = m
	}
	ltgrpc.grpcLoaderServer.SetMethods(ms)
	return nil
}

func (ltgrpc *LoaderTypeGRPC) SetPluginMethodHandler(handler func(map[string]func([]*Arg) ([]*Arg, error))) error {
	ltgrpc.grpcLoaderServer.SetPluginMethodHandler(func(pluginMethods map[string]func(string, []*grpcloader.Arg) ([]*grpcloader.Arg, error)) { // get plugin method map
		// 创建适配后的方法映射
		adaptedMethods := make(map[string]func([]*Arg) ([]*Arg, error))

		for name, pluginMethod := range pluginMethods {
			// 捕获当前方法名和原始方法
			currentName := name
			currentMethod := pluginMethod

			// 适配方法签名
			adaptedMethods[currentName] = func(args []*Arg) ([]*Arg, error) {
				// 转换输入参数
				inArgs := make([]*grpcloader.Arg, len(args))
				for i, arg := range args {
					inArgs[i] = &grpcloader.Arg{
						Name: arg.Name,
						Type: arg.Type,
						Arg:  arg.Data, // Data -> Arg
					}
				}

				// 调用原始插件方法
				outArgs, err := currentMethod(currentName, inArgs)
				if err != nil {
					return nil, err
				}

				// 转换输出参数
				results := make([]*Arg, len(outArgs))
				for i, arg := range outArgs {
					results[i] = &Arg{
						Name: arg.Name,
						Type: arg.Type,
						Data: arg.Arg, // Arg -> Data
					}
				}
				return results, nil
			}
		}
		handler(adaptedMethods) // call plugin method handler
	})
	return nil
}

func (ltgrpc *LoaderTypeGRPC) SetUnregisterMethodsHandler(handler func([]string)) error {
	ltgrpc.grpcLoaderServer.SetUnregisterMethodsHandler(handler)
	return nil
}

func (ltgrpc *LoaderTypeGRPC) UnregisterMethods(methods []string) error {
	for _, method := range methods {
		ltgrpc.publicMethods.Delete(method)
	}
	return nil
}

// this is used for call local method from gRPC
func (ltgrpc *LoaderTypeGRPC) CallMethod(method string, args []*grpcloader.Arg) ([]*grpcloader.Arg, error) {
	// parse agrs
	parsedArgs := []*Arg{}
	for _, arg := range args {
		parsedArgs = append(parsedArgs, &Arg{
			Name: arg.Name,
			Type: arg.Type,
			Data: arg.Arg,
		})
	}
	fA, ok := ltgrpc.publicMethods.Load(method)
	if !ok {
		return nil, ErrMethodNotFound
	}
	f, ok := fA.(func([]*Arg) ([]*Arg, error))
	if !ok {
		return nil, ErrMethodNotFound
	}
	rt, err := f(parsedArgs)
	if err != nil {
		return nil, err
	}
	rtArgs := []*grpcloader.Arg{}
	for _, arg := range rt {
		rtArgs = append(rtArgs, &grpcloader.Arg{
			Name: arg.Name,
			Type: arg.Type,
			Arg:  arg.Data,
		})
	}
	return rtArgs, err
}

func (ltgrpc *LoaderTypeGRPC) CallPluginMethod(method string, args []*Arg) ([]*Arg, error) {
	inArgs := []*grpcloader.Arg{}
	for _, arg := range args {
		inArgs = append(inArgs, &grpcloader.Arg{
			Name: arg.Name,
			Type: arg.Type,
			Arg:  arg.Data,
		})
	}
	returnArgs, err := ltgrpc.grpcLoaderServer.CallClientMethod(method, inArgs)
	if err != nil {
		return nil, err
	}
	// parse return args
	parsedArgs := []*Arg{}
	for _, arg := range returnArgs {
		parsedArgs = append(parsedArgs, &Arg{
			Name: arg.Name,
			Type: arg.Type,
			Data: arg.Arg,
		})
	}
	return parsedArgs, nil
}

func (ltgrpc *LoaderTypeGRPC) Unload() error {
	if ltgrpc.grpcServer != nil {
		ltgrpc.grpcServer.GracefulStop()
	}
	ltgrpc.cancle()
	ltgrpc.publicMethods.Clear()
	ltgrpc.grpcLoaderServer = nil
	return nil
}

// simply call plugin method to unload
func (ltgrpc *LoaderTypeGRPC) UnloadPlugin(pluginID string) error {
	ltgrpc.CallPluginMethod("Unload", []*Arg{})
	return nil
}

func (ltgrpc *LoaderTypeGRPC) ID() string {
	return ltgrpc.id
}

func (ltgrpc *LoaderTypeGRPC) SetID(id string) {
	ltgrpc.id = id
}
