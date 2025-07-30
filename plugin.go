package main

import (
	"LiteBlog/utils/plugins"
	"fmt"
)

var (
	pluginManager = plugins.NewPluginManager()
)

func InitPlugins() {
	loader := &plugins.LoaderTypeGRPC{
		ListenerAddress: "127.0.0.1:10800",
	}
	methodsMap := map[string]func(args []*plugins.Arg) ([]*plugins.Arg, error){
		"AddRenderMap":    AddRenderMap,
		"GetRenderMap":    GetRenderMap,
		"DeleteRenderMap": DeleteRenderMap,
		"AddHook":         AddHook,
		"DeleteHook":      DeleteHook,
	}
	pluginManager.RegisterMethods(methodsMap)
	loaderId, err := pluginManager.RegisterLoader(loader)
	if err != nil {
		Log(3, fmt.Sprintf("Register plugin loader failed: %s", err))
	}
	Log(2, fmt.Sprintf("Register plugin loader success, id: %s", loaderId))
}

// plugin interface:
func AddRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	class := ""
	key := ""
	data := []byte{}
	ret := []*plugins.Arg{}
	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = string(arg.Data)
		case "key":
			key = string(arg.Data)
		case "data":
			data = arg.Data
		}
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
	}
	ret = append(ret, &plugins.Arg{
		Name: "success",
		Data: []byte("true"),
	})
	return ret, nil
}

// plugin interface:
func GetRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	class := ""
	key := ""
	ret := []*plugins.Arg{}
	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = string(arg.Data)
		case "key":
			key = string(arg.Data)
		}
	}
	switch class {
	case "rendered":
		RenderedMapLocker.RLock()
		data, ok := RenderedMap[key]
		RenderedMapLocker.RUnlock()
		if ok {
			ret = append(ret, &plugins.Arg{
				Name: "data",
				Data: data,
			})
		}
	case "global":
		GlobalMapLocker.RLock()
		data, ok := GlobalMap[key]
		GlobalMapLocker.RUnlock()
		if ok {
			ret = append(ret, &plugins.Arg{
				Name: "data",
				Data: data,
			})
		}
	}
	return ret, nil
}

// plugin interface:
func DeleteRenderMap(args []*plugins.Arg) ([]*plugins.Arg, error) {
	class := ""
	key := ""
	ret := []*plugins.Arg{}
	for _, arg := range args {
		switch arg.Name {
		case "class":
			class = string(arg.Data)
		case "key":
			key = string(arg.Data)
		}
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
	}
	ret = append(ret, &plugins.Arg{
		Name: "success",
		Data: []byte("true"),
	})
	return ret, nil
}

// plugin interface:
// Hook class like onRequest ...
func AddHook(args []*plugins.Arg) ([]*plugins.Arg, error) {
	ret := []*plugins.Arg{}
	hook_name := ""
	hook_class := ""
	callback_name := ""
	result := "false"
	for _, arg := range args {
		switch arg.Name {
		case "name":
			hook_name = string(arg.Data)
		case "class":
			hook_class = string(arg.Data)
		case "callback":
			callback_name = string(arg.Data)
		}
	}
	switch hook_class {
	case "onRequest":
		Log(2, fmt.Sprintf("add request hook: %s", hook_name))
		RequestHookRadixTree, _, _ = RequestHookRadixTree.Insert([]byte(hook_name), []byte(callback_name))
		result = "true"
	}
	ret = append(ret, &plugins.Arg{
		Name: "success",
		Data: []byte(result),
	})

	return ret, nil
}

// plugin interface:
func DeleteHook(args []*plugins.Arg) ([]*plugins.Arg, error) {
	ret := []*plugins.Arg{}
	hook_name := ""
	hook_class := ""
	result := "false"
	for _, arg := range args {
		switch arg.Name {
		case "name":
			hook_name = string(arg.Data)
		case "class":
			hook_class = string(arg.Data)
		}
	}
	switch hook_class {
	case "onRequest":
		Log(2, fmt.Sprintf("delete request hook: %s", hook_name))
		RequestHookRadixTree, _, _ = RequestHookRadixTree.Delete([]byte(hook_name))
		result = "true"
	}
	ret = append(ret, &plugins.Arg{
		Name: "success",
		Data: []byte(result),
	})

	return ret, nil
}
