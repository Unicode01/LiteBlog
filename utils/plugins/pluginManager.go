// This is PluginManager which used to manage plugins.
// 这是插件管理器的初步代码,用于管理插件加载器和插件公共方法.
package plugins

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"maps"
)

var (
	ErrNotFound = errors.New("plugin not found")
)

type PluginManager struct {
	loaders       map[string]LoaderType
	publicMethods map[string]func(args []*Arg) ([]*Arg, error)
	pluginMethods map[string]func(args []*Arg) ([]*Arg, error)
}

type Arg struct {
	Name string
	Type string
	Data []byte
}

func NewPluginManager() *PluginManager {
	pm := &PluginManager{}
	pm.publicMethods = make(map[string]func(args []*Arg) ([]*Arg, error))
	pm.pluginMethods = make(map[string]func(args []*Arg) ([]*Arg, error))
	pm.loaders = make(map[string]LoaderType)
	return pm
}

// RegisterLoader register a plugin loader
func (pm *PluginManager) RegisterLoader(loader LoaderType) (id string, err error) {
	// init plugin
	err = loader.Init()
	if err != nil {
		return "", err
	}
	// load plugin
	err = loader.Load()
	if err != nil {
		return "", err
	}
	tmpID := pm.newID()
	loader.SetID(tmpID)
	pm.loaders[tmpID] = loader
	// register public methods
	loader.RegisterMethods(pm.publicMethods)
	loader.SetPluginMethodHandler(pm.registerPluginMethods)
	loader.SetUnregisterMethodsHandler(pm.unregisterPluginMethods)
	return tmpID, nil
}

func (pm *PluginManager) UnregisterLoader(id string) (err error) {
	loader, ok := pm.loaders[id]
	if !ok {
		return ErrNotFound
	}
	err = loader.Unload()
	if err != nil {
		return err
	}
	delete(pm.loaders, id)
	return nil
}

// RegisterMethods register public methods
func (pm *PluginManager) RegisterMethods(methods map[string]func(args []*Arg) ([]*Arg, error)) {
	// merge public methods
	maps.Copy(pm.publicMethods, methods)
}

func (pm *PluginManager) registerPluginMethods(methods map[string]func(args []*Arg) ([]*Arg, error)) {
	// merge plugin methods
	maps.Copy(pm.pluginMethods, methods)
}

// UnregisterMethods unregister public methods by name
func (pm *PluginManager) UnregisterMethods(name []string) {
	for _, n := range name {
		delete(pm.publicMethods, n)
	}
	for _, loader := range pm.loaders {
		loader.UnregisterMethods(name)
	}
}

func (pm *PluginManager) unregisterPluginMethods(name []string) {
	for _, n := range name {
		delete(pm.pluginMethods, n)
	}
	if len(pm.pluginMethods) == 0 {
		pm.pluginMethods = make(map[string]func(args []*Arg) ([]*Arg, error))
	}
}

// Call the public method of the plugin
func (pm *PluginManager) CallPluginMethod(method string, args []*Arg) ([]*Arg, error) {
	pluginMethod, ok := pm.pluginMethods[method]
	if !ok {
		return nil, errors.New("plugin method not found")
	}
	return pluginMethod(args)
}

func (pm *PluginManager) newID() string {
	// TODO: generate unique id
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	for _, loader := range pm.loaders {
		if loader.ID() == id {
			return pm.newID()
		}
	}
	return id
}
