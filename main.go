package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
)

var (
	Config AllConfig
)

func main() {
	go ExitListener()
	Config = ReadConfig()
	SetVarToGlobalMap()
	ReadGolbalConfig()
	BackupConfigures()

	ReadFlag()

	AutoAddListener()
	err := InitNetManager(&Config.ServerCfg)
	if err != nil {
		panic(err)
	}
	select {}
}

func ReadFlag() {
	var generateStatic bool
	flag.BoolVar(&generateStatic, "static", false, "generate static files")
	flag.Parse()
	if generateStatic {
		RenderStatic()
		Log(1, "Static files generated")
		os.Exit(0)
	}

}

func ExitListener() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	sig := <-c
	Log(1, "Exiting with signal: "+sig.String())
	CloseLogger()
	fireWall.SaveRules()
	deliverManager.Shutdown()
	os.Exit(0)
}
