package main

import (
	"fmt"
	"os"

	"gitlab.com/sniffdogsniff/sds"
	"gitlab.com/sniffdogsniff/util/logging"
	"gitlab.com/sniffdogsniff/webui"
)

func showHelp() {
	fmt.Println("SniffDogSniff v0.1")
	fmt.Println("\t-c or --config\tSpecify config file location")
	os.Exit(0)
}

func parseArgs() (string, bool) {
	cfgFilePath := "./config.ini"
	runAsDaemon := false

	for i := 0; i < len(os.Args); i++ {
		switch arg := os.Args[i]; arg {
		case "-c":
		case "--config":
			cfgFilePath = os.Args[i+1]
			i++
			break
		case "-d":
		case "--daemon":
			runAsDaemon = true
			break
		case "-h":
		case "--help":
			showHelp()
			break
		}
	}
	return cfgFilePath, runAsDaemon
}

func main() {
	cfgFilePath, _ := parseArgs()

	confs := sds.NewSdsConfig(cfgFilePath)

	logging.InitLogging(logging.TRACE)

	node := sds.InitNode(confs)

	p2pServer := sds.InitNodeServer(&node)
	go p2pServer.Serve(confs.NodePeerInfo.Address)

	webServer := webui.InitSdsWebServer(&node)
	go webServer.ServeWebUi(confs.WebServiceBindAddress)

	node.SyncWithPeers()

}
