package plugins

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/dop251/goja"
)

//go:embed javascriptLoader/init.js
var InitJSCode string

type LoaderTypeJS struct {
	Loader
	jsvmPool      *sync.Pool
	ctx           context.Context
	cancle        context.CancelFunc
	publicMethods sync.Map // map[string]func([]*Arg) ([]*Arg, error)
}

func (loader *LoaderTypeJS) Init() error {
	loader.ctx, loader.cancle = context.WithCancel(context.Background())
	loader.jsvmPool = &sync.Pool{
		New: func() interface{} {
			vm := goja.New()
			vm.Set("loaderId", loader.id)
			vm.Set("cancel", func() {
				loader.cancle()
			})
			v, err := vm.RunString(InitJSCode)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Print(v.ToString())
			return vm
		},
	}
	return nil
}

func (loader *LoaderTypeJS) Load() error {
	return nil
}

func (loader *LoaderTypeJS) RegisterMethods(methods map[string]func([]*Arg) ([]*Arg, error)) error {
	return nil
}

func (loader *LoaderTypeJS) SetPluginMethodHandler(handler func(map[string]func([]*Arg) ([]*Arg, error))) error {
	return nil
}

func (loader *LoaderTypeJS) SetUnregisterMethodsHandler(handler func([]string)) error {
	return nil
}

func (loader *LoaderTypeJS) UnregisterMethods(methods []string) error {
	return nil
}

func (loader *LoaderTypeJS) Unload() error {
	return nil
}

func (Loader *LoaderTypeJS) UnloadPlugin(pluginID string) error {
	return nil
}

func (loader *LoaderTypeJS) CallPluginMethod(method string, args []*Arg) ([]*Arg, error) {
	return nil, nil
}

func (loader *LoaderTypeJS) ID() string {
	return loader.id
}

func (loader *LoaderTypeJS) SetID(id string) {
	loader.id = id
}
