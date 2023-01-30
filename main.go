package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
		case "-c", "--config":
			cfgFilePath = os.Args[i+1]
			i++
			break
		case "-d", "--daemon":
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

func shutdownHook(configs sds.SdsConfig, p sds.Peer) {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	go func() {
		for {
			s := <-sigchnl
			if s == syscall.SIGINT {
				if configs.ServiceSettings.CreateHiddenService {
					sds.RemoveHiddenService(configs.ServiceSettings, p)
				}
				logging.LogInfo("Shutting down...")
				os.Exit(0)
			}
		}
	}()
}

func main() {
	logging.InitLogging(logging.TRACE)
	cfgFilePath, _ := parseArgs()

	confs := sds.NewSdsConfig(cfgFilePath)

	nodeServerBindAddress := confs.ServiceSettings.PeerInfo.Address

	if confs.ServiceSettings.CreateHiddenService {
		confs.ServiceSettings.PeerInfo = sds.CreateHiddenService(confs.ServiceSettings)
	}

	node := sds.GetNodeInstance(confs)

	p2pServer := sds.InitNodeServer(node)
	go p2pServer.Serve(nodeServerBindAddress)

	webServer := webui.InitSdsWebServer(node)
	go webServer.ServeWebUi(confs.WebServiceBindAddress)

	logging.LogInfo("SniffDogSniff started press CTRL-C to stop")

	shutdownHook(confs, node.SelfPeer)

	node.SyncWithPeers()

}
