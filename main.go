package main

import (
	"flag"
	"os"
)

var (
	Config AllConfig
)

func main() {
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
	if Config.BackupCfg.Enabled {
		BackupNow()
	}
	select {}
}

func ReadFlag() {
	var generateStatic bool
	flag.BoolVar(&generateStatic, "static", false, "host to listen on")
	flag.Parse()
	if generateStatic {
		RenderStatic()
		Log(1, "Static files generated")
		os.Exit(0)
	}

}
