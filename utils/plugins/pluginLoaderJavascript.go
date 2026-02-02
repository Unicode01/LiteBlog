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
	PluginDir                           string        // 插件目录，默认 "plugins"
	InitDelay                           time.Duration // 初始化延迟，默认 2 秒
	jsvmMap                             sync.Map      // map[pluginID string]*goja.Runtime
	ctx                                 context.Context
	cancle                              context.CancelFunc
	publicMethods                       sync.Map // map[string]func([]*Arg) ([]*Arg, error)
	pluginMethods                       sync.Map // map[string]struct{vm *goja.Runtime, method goja.Callable}
	registerPluginMethodsManagerHandler func(map[string]func([]*Arg) ([]*Arg, error))
	unregisterMethodsHandler            func([]string) // 注销方法时的回调
}

func (loader *LoaderTypeJS) Init() error {
	loader.ctx, loader.cancle = context.WithCancel(context.Background())
	loader.jsvmMap = sync.Map{}
	loader.publicMethods = sync.Map{}
	loader.pluginMethods = sync.Map{}

	// 设置默认值
	if loader.PluginDir == "" {
		loader.PluginDir = "plugins"
	}
	if loader.InitDelay == 0 {
		loader.InitDelay = 2 * time.Second
	}
	return nil
}

func (loader *LoaderTypeJS) Load() error {
	go func() {
		time.Sleep(loader.InitDelay)
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
	loader.unregisterMethodsHandler = handler
	return nil
}

func (loader *LoaderTypeJS) UnregisterMethods(methods []string) error {
	for _, methodName := range methods {
		loader.publicMethods.Delete(methodName)
	}
	return nil
}

func (loader *LoaderTypeJS) Unload() error {
	// 取消 context
	if loader.cancle != nil {
		loader.cancle()
	}

	// 收集所有插件方法名用于通知管理器
	methodNames := make([]string, 0)
	loader.pluginMethods.Range(func(key, value interface{}) bool {
		methodNames = append(methodNames, key.(string))
		return true
	})

	// 通知管理器注销这些方法
	if loader.unregisterMethodsHandler != nil && len(methodNames) > 0 {
		loader.unregisterMethodsHandler(methodNames)
	}

	// 清理所有 JS 虚拟机
	loader.jsvmMap.Range(func(key, value interface{}) bool {
		loader.jsvmMap.Delete(key)
		return true
	})

	// 清理方法映射
	loader.pluginMethods.Clear()
	loader.publicMethods.Clear()

	utils.Log(1, "JavaScript plugin loader unloaded successfully")
	return nil
}

func (loader *LoaderTypeJS) UnloadPlugin(pluginID string) error {
	// 检查插件是否存在
	vmA, ok := loader.jsvmMap.Load(pluginID)
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	// 尝试调用插件的 onUnload 函数（如果存在）
	if vm, ok := vmA.(*goja.Runtime); ok {
		if onUnload := vm.Get("onUnload"); onUnload != nil && !goja.IsUndefined(onUnload) {
			if callable, ok := goja.AssertFunction(onUnload); ok {
				callable(goja.Undefined())
			}
		}
	}

	// 收集该插件注册的方法
	methodsToRemove := make([]string, 0)
	loader.pluginMethods.Range(func(key, value interface{}) bool {
		methodInfo := value.(struct {
			vm     *goja.Runtime
			method goja.Callable
		})
		// 检查方法是否属于该插件的 VM
		if vmA == methodInfo.vm {
			methodsToRemove = append(methodsToRemove, key.(string))
		}
		return true
	})

	// 删除插件方法
	for _, methodName := range methodsToRemove {
		loader.pluginMethods.Delete(methodName)
	}

	// 通知管理器注销这些方法
	if loader.unregisterMethodsHandler != nil && len(methodsToRemove) > 0 {
		loader.unregisterMethodsHandler(methodsToRemove)
	}

	// 删除 VM
	loader.jsvmMap.Delete(pluginID)

	utils.Log(1, fmt.Sprintf("Plugin %s unloaded successfully", pluginID))
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
	files, err := os.ReadDir(loader.PluginDir)
	if err != nil {
		utils.Log(3, fmt.Sprintf("load plugins from %s failed: %s", loader.PluginDir, err.Error()))
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
	pluginPath := path.Join(loader.PluginDir, pluginName+".js")
	// create new runtime
	vm := goja.New()
	// random id
	pluginId := loader.NewPluginID()

	// 保存 VM 到映射中
	loader.jsvmMap.Store(pluginId, vm)

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

	// 捕获当前 VM 用于闭包
	currentVM := vm
	currentPluginName := pluginName

	vm.Set("registerMethods", func(methods []string) {
		// store plugin methods
		methodMap := make(map[string]func([]*Arg) ([]*Arg, error))
		for _, methodName := range methods {
			currentMethodName := methodName // 捕获变量
			tmpf, ok := goja.AssertFunction(currentVM.Get(currentMethodName))
			if !ok {
				utils.Log(3, fmt.Sprintf("Plugin %s: method %s not found", currentPluginName, currentMethodName))
				return
			}
			loader.pluginMethods.Store(currentMethodName, struct {
				vm     *goja.Runtime
				method goja.Callable
			}{
				vm:     currentVM,
				method: tmpf,
			})
			methodMap[currentMethodName] = func(args []*Arg) ([]*Arg, error) {
				return loader.pluginMethodProxy(currentMethodName, args)
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
		loader.jsvmMap.Delete(pluginId) // 加载失败时清理
		utils.Log(3, fmt.Sprintf("init.js load failed: %s", err.Error()))
		return err
	}

	// load plugin
	pluginJS, err := os.ReadFile(pluginPath)
	if err != nil {
		loader.jsvmMap.Delete(pluginId) // 加载失败时清理
		utils.Log(3, fmt.Sprintf("Plugin %s: load failed: %s", pluginName, err.Error()))
		return err
	}
	_, err = vm.RunString(string(pluginJS))
	if err != nil {
		loader.jsvmMap.Delete(pluginId) // 加载失败时清理
		utils.Log(3, fmt.Sprintf("Plugin %s: load failed: %s", pluginName, err.Error()))
		return err
	}

	utils.Log(1, fmt.Sprintf("Plugin %s loaded successfully with ID %s", pluginName, pluginId))
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
		// 优化：对于 json 类型，如果 Data 是 []byte，转换为 string
		// 这样 JavaScript 端就不需要再做字节数组转换，提升性能
		data := arg.Data
		if arg.Type == "json" {
			if bytes, ok := data.([]byte); ok {
				data = string(bytes)
			}
		}

		argarry[i] = map[string]any{
			"Name": arg.Name,
			"Type": arg.Type,
			"Data": data,
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
			Name: utils.GetStringSafe(rA["Name"]),
			Type: utils.GetStringSafe(rA["Type"]),
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

// 类型转换工具函数已移至 utils 包: utils.GetStringSafe, utils.GetBytesSafe, utils.GetIntSafe
