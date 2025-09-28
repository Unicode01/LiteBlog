// This is PluginManager which used to manage plugins.
// 这是插件管理器的初步代码,用于管理插件加载器和插件公共方法.
package plugins

import (
	"LiteBlog/utils"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNotFound = errors.New("plugin not found")
)

type PluginManager struct {
	loaders           map[string]LoaderType
	publicMethods     sync.Map //map[string]func(args []*Arg) ([]*Arg, error)
	pluginMethods     sync.Map //map[string]func(args []*Arg) ([]*Arg, error)
	managerLoaderLock sync.Mutex
}

type Arg struct {
	Name string
	Type string
	Data any
}

func NewPluginManager() *PluginManager {
	pm := &PluginManager{}
	pm.loaders = make(map[string]LoaderType)
	return pm
}

// RegisterLoader register a plugin loader
func (pm *PluginManager) RegisterLoader(loader LoaderType) (id string, err error) {
	pm.managerLoaderLock.Lock()
	defer pm.managerLoaderLock.Unlock()
	// init plugin
	err = loader.Init()
	if err != nil {
		return "", err
	}
	tmpID := pm.newID()
	loader.SetID(tmpID)
	// load plugin
	err = loader.Load()
	if err != nil {
		return "", err
	}
	pm.loaders[tmpID] = loader
	// register public methods
	methodsMap := make(map[string]func(args []*Arg) ([]*Arg, error))
	pm.publicMethods.Range(func(k, v any) bool {
		methodsMap[k.(string)] = v.(func(args []*Arg) ([]*Arg, error))
		return true
	})
	loader.RegisterMethods(methodsMap)
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
	for n, m := range methods {
		pm.publicMethods.Store(n, m)
	}
}

func (pm *PluginManager) registerPluginMethods(methods map[string]func(args []*Arg) ([]*Arg, error)) {
	// merge plugin methods
	for n, m := range methods {
		_, ok := pm.pluginMethods.Load(n)
		if ok {
			utils.Log(2, fmt.Sprintf("plugin method '%s' has been registered!", n))
			continue
		}
		pm.pluginMethods.Store(n, m)
	}
}

func (pm *PluginManager) GetPluginMethods() []string {
	methods := make([]string, 0)
	pm.pluginMethods.Range(func(k, v any) bool {
		methods = append(methods, k.(string))
		return true
	})
	return methods
}

// UnregisterMethods unregister public methods by name
func (pm *PluginManager) UnregisterMethods(name []string) {
	for _, n := range name {
		pm.publicMethods.Delete(n)
	}
	for _, loader := range pm.loaders {
		loader.UnregisterMethods(name)
	}
}

func (pm *PluginManager) unregisterPluginMethods(name []string) {
	for _, n := range name {
		pm.pluginMethods.Delete(n)
	}
}

// Call the public method of the plugin
func (pm *PluginManager) CallPluginMethod(method string, args []*Arg) ([]*Arg, error) {
	pluginMethodA, ok := pm.pluginMethods.Load(method)
	if !ok {
		return nil, ErrMethodNotFound
	}
	pluginMethod, ok := pluginMethodA.(func(args []*Arg) ([]*Arg, error))
	if !ok {
		return nil, ErrMethodNotFound
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
