package plugins

import (
	"LiteBlog/utils"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/dop251/goja"
)

//go:embed javascriptLoader/init.js
var InitJSCode string

var (
	version = "0.0.1"
)

type LoaderTypeJS struct {
	Loader
	LastError                           error
	jsvmMap                             sync.Map // map[pluginID string]*goja.Runtime
	ctx                                 context.Context
	cancle                              context.CancelFunc
	publicMethods                       sync.Map // map[string]func([]*Arg) ([]*Arg, error)
	pluginMethods                       sync.Map // map[string]struct{vm *goja.Runtime, method goja.Callable}
	registerPluginMethodsManagerHandler func(map[string]func([]*Arg) ([]*Arg, error))
}

func (loader *LoaderTypeJS) Init() error {
	loader.ctx, loader.cancle = context.WithCancel(context.Background())
	loader.jsvmMap = sync.Map{}
	loader.publicMethods = sync.Map{}
	loader.pluginMethods = sync.Map{}
	return nil
}

func (loader *LoaderTypeJS) Load() error {
	go func() {
		time.Sleep(2 * time.Second)
		loader.LoadAllPlugins()
	}()
	return nil
}

func (loader *LoaderTypeJS) RegisterMethods(methods map[string]func([]*Arg) ([]*Arg, error)) error {
	for methodName, methodFunc := range methods {
		loader.publicMethods.Store(methodName, methodFunc)
	}

	return nil
}

// called when plugin load, set plugin method handler
func (loader *LoaderTypeJS) SetPluginMethodHandler(handler func(map[string]func([]*Arg) ([]*Arg, error))) error {
	loader.registerPluginMethodsManagerHandler = handler
	return nil
}

func (loader *LoaderTypeJS) SetUnregisterMethodsHandler(handler func([]string)) error {
	return nil
}

func (loader *LoaderTypeJS) UnregisterMethods(methods []string) error {
	for _, methodName := range methods {
		loader.publicMethods.Delete(methodName)
	}
	return nil
}

func (loader *LoaderTypeJS) Unload() error {
	return nil
}

func (Loader *LoaderTypeJS) UnloadPlugin(pluginID string) error {
	return nil
}

func (loader *LoaderTypeJS) CallPluginMethod(method string, args []*Arg) ([]*Arg, error) {
	return loader.pluginMethodProxy(method, args)
}

func (loader *LoaderTypeJS) ID() string {
	return loader.id
}

func (loader *LoaderTypeJS) SetID(id string) {
	loader.id = id
}

func (loader *LoaderTypeJS) LoadAllPlugins() error {
	files, err := os.ReadDir("plugins")
	if err != nil {
		utils.Log(3, fmt.Sprintf("load plugins failed: %s", err.Error()))
		return err
	}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".js" {
			pluginName := file.Name()[:len(file.Name())-3] // remove .js
			loader.LoadPlugin(pluginName)
		}
	}
	return nil
}

func (loader *LoaderTypeJS) LoadPlugin(pluginName string) error {
	pluginPath := path.Join("plugins", pluginName+".js")
	// create new runtime
	vm := goja.New()
	// random id
	pluginId := loader.NewPluginID()
	// set env
	vm.Set("loaderVersion", version)
	vm.Set("pluginId", pluginId)
	vm.Set("pluginName", pluginName)
	vm.Set("pluginDirPath", pluginPath)
	vm.Set("log", func(level int, msg string) {
		utils.Log(level, fmt.Sprintf("Plugin %s: %s", pluginName, msg))
	})
	loader.publicMethods.Range(func(key, value interface{}) bool {
		vm.Set(key.(string), value)
		return true
	})
	vm.Set("registerMethods", func(methods []string) {
		// store plugin methods
		methodMap := make(map[string]func([]*Arg) ([]*Arg, error))
		for _, methodName := range methods {
			tmpf, ok := goja.AssertFunction(vm.Get(methodName))
			if !ok {
				utils.Log(3, fmt.Sprintf("Plugin %s: method %s not found", pluginName, methodName))
				return
			}
			loader.pluginMethods.Store(methodName, struct {
				vm     *goja.Runtime
				method goja.Callable
			}{
				vm:     vm,
				method: tmpf,
			})
			methodMap[methodName] = func(args []*Arg) ([]*Arg, error) {
				return loader.pluginMethodProxy(methodName, args)
			}
		}
		// call plugin method handler
		loader.registerPluginMethodsHandler(methodMap)
	})
	vm.Set("getPublicMethods", func() []string {
		var methods []string
		loader.publicMethods.Range(func(key, value interface{}) bool {
			methods = append(methods, key.(string))
			return true
		})
		return methods
	})
	// load init.js
	_, err := vm.RunString(InitJSCode)
	if err != nil {
		utils.Log(3, fmt.Sprintf("init.js load failed: %s", err.Error()))
		return err
	}
	// load plugin
	pluginJS, err := os.ReadFile(pluginPath)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Plugin %s: load failed: %s", pluginName, err.Error()))
		return err
	}
	_, err = vm.RunString(string(pluginJS))
	if err != nil {
		utils.Log(3, fmt.Sprintf("Plugin %s: load failed: %s", pluginName, err.Error()))
		return err
	}
	return nil
}

// plugin caller
func (loader *LoaderTypeJS) registerPluginMethodsHandler(methods map[string]func([]*Arg) ([]*Arg, error)) error {
	loader.registerPluginMethodsManagerHandler(methods)
	return nil
}

func (loader *LoaderTypeJS) pluginMethodProxy(method string, args []*Arg) (rt []*Arg, err error) {
	// get method
	methodFunc, ok := loader.pluginMethods.Load(method)
	if !ok {
		utils.Log(3, fmt.Sprintf("Plugin %s: method %s not found", loader.id, method))
		return nil, fmt.Errorf("method %s not found", method)
	}
	methodFuncTyped := methodFunc.(struct {
		vm     *goja.Runtime
		method goja.Callable
	})
	// adapt args
	argarry := make([]interface{}, len(args))
	for i, arg := range args {
		argarry[i] = map[string]any{
			"Name": arg.Name,
			"Type": arg.Type,
			"Data": arg.Data,
		}
	}
	gojaArgsArray := methodFuncTyped.vm.NewArray(argarry...)

	// call method
	rtb, err := methodFuncTyped.method(methodFuncTyped.vm.ToValue(nil), gojaArgsArray)
	if err != nil {
		return nil, err
	}
	// convert result
	rt = make([]*Arg, len(rtb.Export().([]interface{})))
	for i, r := range rtb.Export().([]interface{}) {
		rA := r.(map[string]interface{})
		rt[i] = &Arg{
			Name: getString_safe(rA["Name"]),
			Type: getString_safe(rA["Type"]),
			Data: rA["Data"],
		}
	}
	return rt, nil
}

func (loader *LoaderTypeJS) NewPluginID() string {
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	if _, ok := loader.jsvmMap.Load(id); ok {
		return loader.NewPluginID()
	}
	return id
}

func getString_safe(data any) string {
	switch v := data.(type) {
	case string:
		return v
	case []uint8:
		return string(v)
	default:
		return fmt.Sprintf("%v", data)
	}
}

func getBytes_safe(data any) []byte {
	switch v := data.(type) {
	case string:
		return []byte(v)
	case []uint8:
		return v
	default:
		return fmt.Appendf(nil, "%v", data)
	}
}

func getInt_safe(data any) int {
	switch v := data.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}