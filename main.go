package main

var (
	Config AllConfig
)

func main() {
	Config = ReadConfig()
	SetVarToGlobalMap()
	ReadGolbalConfig()
	BackupConfigures()
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
