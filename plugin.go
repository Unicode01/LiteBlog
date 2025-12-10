package main

import (
	"errors"
	"fmt"
	"time"

	utils "LiteBlog/utils"
	"LiteBlog/utils/plugins"
)

var (
	pluginManager = plugins.NewPluginManager()
)

func InitPlugins() {
	cfg := Config.PluginCfg

	// 注册公共方法（所有加载器共享）
	methodsMap := map[string]func(args []*plugins.Arg) ([]*plugins.Arg, error){
		"AddRenderMap":        AddRenderMap,
		"GetRenderMap":        GetRenderMap,
		"DeleteRenderMap":     DeleteRenderMap,
		"AddHook":             AddHook,
		"DeleteHook":          DeleteHook,
		"AddRouteListener":    AddRouteListener,
		"DeleteRouteListener": DeleteRouteListener,
	}
	pluginManager.RegisterMethods(methodsMap)

	// 注册 gRPC 插件加载器
	if cfg.GRPCConfig.Enabled {
		grpcLoader := &plugins.LoaderTypeGRPC{
			ListenerAddress: cfg.GRPCConfig.ListenerAddress,
			AccessKey:       cfg.GRPCConfig.AccessKey,
		}
		// 设置命令超时时间
		if cfg.GRPCConfig.CommandTimeout > 0 {
			grpcLoader.CommandTimeout = time.Duration(cfg.GRPCConfig.CommandTimeout) * time.Second
		}
		loaderId, err := pluginManager.RegisterLoader(grpcLoader)
		if err != nil {
			utils.Log(3, fmt.Sprintf("Register gRPC plugin loader failed: %s", err))
		} else {
			accessKeyStatus := "disabled"
			if cfg.GRPCConfig.AccessKey != "" {
				accessKeyStatus = "enabled"
			}
			utils.Log(1, fmt.Sprintf("gRPC plugin loader registered, id: %s, address: %s, access_key: %s", loaderId, cfg.GRPCConfig.ListenerAddress, accessKeyStatus))
		}
	} else {
		utils.Log(1, "gRPC plugin loader is disabled")
	}

	// 注册 JavaScript 插件加载器
	if cfg.JSConfig.Enabled {
		jsLoader := &plugins.LoaderTypeJS{
			PluginDir: cfg.JSConfig.PluginDir,
			InitDelay: time.Duration(cfg.JSConfig.InitDelay) * time.Second,
		}
		loaderId, err := pluginManager.RegisterLoader(jsLoader)
		if err != nil {
			utils.Log(3, fmt.Sprintf("Register JavaScript plugin loader failed: %s", err))
		} else {
			utils.Log(1, fmt.Sprintf("JavaScript plugin loader registered, id: %s, plugin_dir: %s", loaderId, cfg.JSConfig.PluginDir))
		}
	} else {
		utils.Log(1, "JavaScript plugin loader is disabled")
	}
}

// plugin interface:
func AddRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var class, key string
	var data []byte

	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = utils.GetStringSafe(arg.Data)
		case "key":
			key = utils.GetStringSafe(arg.Data)
		case "data":
			data = utils.GetBytesSafe(arg.Data)
		}
	}

	// 参数验证
	if class == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	switch class {
	case "rendered":
		RenderedMapLocker.Lock()
		RenderedMap[key] = data
		RenderedMapLocker.Unlock()
	case "global":
		GlobalMapLocker.Lock()
		GlobalMap[key] = data
		GlobalMapLocker.Unlock()
	default:
		return nil, fmt.Errorf("unknown class: %s (expected: rendered, global)", class)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// plugin interface:
func GetRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var class, key string

	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = utils.GetStringSafe(arg.Data)
		case "key":
			key = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if class == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	var data []byte
	var found bool

	switch class {
	case "rendered":
		RenderedMapLocker.RLock()
		data, found = RenderedMap[key]
		RenderedMapLocker.RUnlock()
	case "global":
		GlobalMapLocker.RLock()
		data, found = GlobalMap[key]
		GlobalMapLocker.RUnlock()
	default:
		return nil, fmt.Errorf("unknown class: %s (expected: rendered, global)", class)
	}

	if !found {
		return []*plugins.Arg{
			{Name: "found", Type: "bool", Data: false},
		}, nil
	}

	return []*plugins.Arg{
		{Name: "found", Type: "bool", Data: true},
		{Name: "data", Type: "bytes", Data: data},
	}, nil
}

// plugin interface:
func DeleteRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var class, key string

	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = utils.GetStringSafe(arg.Data)
		case "key":
			key = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if class == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if key == "" {
		return nil, errors.New("missing required parameter: key")
	}

	switch class {
	case "rendered":
		RenderedMapLocker.Lock()
		delete(RenderedMap, key)
		RenderedMapLocker.Unlock()
	case "global":
		GlobalMapLocker.Lock()
		delete(GlobalMap, key)
		GlobalMapLocker.Unlock()
	default:
		return nil, fmt.Errorf("unknown class: %s (expected: rendered, global)", class)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// plugin interface:
// Hook class like onRequest ...
// 支持的路由格式：
//   - 精确匹配: /api/users
//   - 参数匹配: /api/users/:id （:前缀表示单段参数）
//   - 通配符匹配: /api/*path （*前缀表示匹配剩余所有路径）
func AddHook(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var hookName, hookClass, callbackName string

	for _, arg := range args {
		switch arg.Name {
		case "name":
			hookName = utils.GetStringSafe(arg.Data)
		case "class":
			hookClass = utils.GetStringSafe(arg.Data)
		case "callback":
			callbackName = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if hookName == "" {
		return nil, errors.New("missing required parameter: name")
	}
	if hookClass == "" {
		return nil, errors.New("missing required parameter: class")
	}
	if callbackName == "" {
		return nil, errors.New("missing required parameter: callback")
	}

	switch hookClass {
	case "onRequest":
		// 检查是否为参数化路由（包含 : 或 *）
		if isParameterizedRoute(hookName) {
			utils.Log(2, fmt.Sprintf("add parameterized request hook: %s -> %s", hookName, callbackName))
			AddParameterizedHook(hookName, callbackName)
		} else {
			utils.Log(2, fmt.Sprintf("add request hook: %s -> %s", hookName, callbackName))
			RequestHookRadixTree, _, _ = RequestHookRadixTree.Insert([]byte(hookName), []byte(callbackName))
		}
	default:
		return nil, fmt.Errorf("unknown hook class: %s (expected: onRequest)", hookClass)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// plugin interface:
func DeleteHook(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var hookName, hookClass string

	for _, arg := range args {
		switch arg.Name {
		case "name":
			hookName = utils.GetStringSafe(arg.Data)
		case "class":
			hookClass = utils.GetStringSafe(arg.Data)
		}
	}

	// 参数验证
	if hookName == "" {
		return nil, errors.New("missing required parameter: name")
	}
	if hookClass == "" {
		return nil, errors.New("missing required parameter: class")
	}

	switch hookClass {
	case "onRequest":
		// 检查是否为参数化路由
		if isParameterizedRoute(hookName) {
			utils.Log(2, fmt.Sprintf("delete parameterized request hook: %s", hookName))
			RemoveParameterizedHook(hookName)
		} else {
			utils.Log(2, fmt.Sprintf("delete request hook: %s", hookName))
			RequestHookRadixTree, _, _ = RequestHookRadixTree.Delete([]byte(hookName))
		}
	default:
		return nil, fmt.Errorf("unknown hook class: %s (expected: onRequest)", hookClass)
	}

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// AddRouteListener 注册路由监听（仅观测，不拦截），支持 request/response/both
// 参数：
//   - route: 路由模式（支持精确/:param/*wildcard）
//   - callback: 插件回调方法名
//   - phase: request/response/both，默认 both
//   - priority: 优先级（可选），数值越大优先级越高，默认 0
func AddRouteListener(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var route, callback, phaseStr string
	var priority int

	for _, arg := range args {
		switch arg.Name {
		case "route":
			route = utils.GetStringSafe(arg.Data)
		case "callback":
			callback = utils.GetStringSafe(arg.Data)
		case "phase":
			phaseStr = utils.GetStringSafe(arg.Data)
		case "priority":
			priority = utils.GetIntSafe(arg.Data)
		}
	}

	if route == "" {
		return nil, errors.New("missing required parameter: route")
	}
	if callback == "" {
		return nil, errors.New("missing required parameter: callback")
	}

	phase := parseListenPhase(phaseStr)
	registerRouteListenerWithPriority(route, callback, phase, priority)

	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// DeleteRouteListener 删除路由监听
// 参数：
//   - route: 路由模式
//   - callback:（可选）指定回调名，仅删除匹配项
func DeleteRouteListener(args []*plugins.Arg) ([]*plugins.Arg, error) {
	var route, callback string

	for _, arg := range args {
		switch arg.Name {
		case "route":
			route = utils.GetStringSafe(arg.Data)
		case "callback":
			callback = utils.GetStringSafe(arg.Data)
		}
	}

	if route == "" {
		return nil, errors.New("missing required parameter: route")
	}

	removeRouteListener(route, callback)
	return []*plugins.Arg{{Name: "success", Type: "bool", Data: true}}, nil
}

// 注意: 类型安全转换函数已移至 utils 包
// 使用: utils.GetStringSafe(), utils.GetBytesSafe(), utils.GetIntSafe()
