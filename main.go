package main

import (
	utils "LiteBlog/utils"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	Config  AllConfig
	Version = "v0.0.7"
)

func main() {
	utils.SetLog()
	go ExitListener()
	Config = ReadConfig()
	SetVarToGlobalMap()
	ReadGolbalConfig()
	BackupConfigures()

	ReadFlag()

	AutoAddListener()
	if Config.PluginCfg.Enabled {
		InitPlugins()
	}
	err := InitNetManager(&Config.ServerCfg)
	if err != nil {
		panic(err)
	}
	select {}
}

func ReadFlag() {
	var generateStatic bool
	var version bool
	flag.BoolVar(&generateStatic, "static", false, "generate static files")
	flag.BoolVar(&version, "version", false, "show version")
	flag.Parse()
	if generateStatic {
		RenderStatic()
		utils.Log(1, "Static files generated")
		os.Exit(0)
	}
	if version {
		fmt.Printf("LiteBlog %s\n", Version)
		os.Exit(0)
	}

}

func ExitListener() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	sig := <-c
	utils.Log(1, "Exiting with signal: "+sig.String())
	utils.CloseLogger()
	fireWall.SaveRules()
	deliverManager.Shutdown()
	os.Exit(0)
}
