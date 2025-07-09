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
		"Unload": PluginUnload,
		"Load":   PluginLoad,
	}
	pluginManager.RegisterMethods(methodsMap)
	loaderId, err := pluginManager.RegisterLoader(loader)
	if err != nil {
		Log(3, fmt.Sprintf("Register plugin loader failed: %s", err))
	}
	Log(2, fmt.Sprintf("Register plugin loader success, id: %s", loaderId))
}

func PluginUnload(args []*plugins.Arg) ([]*plugins.Arg, error) {
	fmt.Printf("PluginUnload called\n")
	return nil, nil
}

func PluginLoad(args []*plugins.Arg) ([]*plugins.Arg, error) {
	fmt.Printf("PluginLoad called\n")
	fmt.Printf("Available methods: %v\n", pluginManager.GetPluginMethods())
	return nil, nil
}
