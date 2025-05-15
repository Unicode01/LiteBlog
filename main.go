package main

var (
	Config AllConfig
)

func main() {
	Config = ReadConfig()
	AutoAddListener()
	err := InitNetManager(&Config.ServerCfg)
	if err != nil {
		panic(err)
	}
	select {}
}
