package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util/logging"
	"github.com/sniffdogsniff/webui"
)

func showHelp() {
	fmt.Println("SniffDogSniff v0.1")
	fmt.Println("\t-c or --config\tSpecify config file location")
	os.Exit(0)
}

func parseArgs() (string, bool, int) {
	cfgFilePath := "./config.ini"
	runAsDaemon := false
	logLevel := logging.INFO

	for i := 0; i < len(os.Args); i++ {
		switch arg := os.Args[i]; arg {
		case "-c", "--config":
			cfgFilePath = os.Args[i+1]
			i++
		case "-d", "--daemon":
			runAsDaemon = true
		case "-h", "--help":
			showHelp()
		case "--log-level":
			logLevel = logging.StrToLogLevel(os.Args[i+1])
			i++
		}
	}
	return cfgFilePath, runAsDaemon, logLevel
}

func shutdownHook(configs sds.SdsConfig, node *sds.LocalNode) {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	go func() {
		for {
			s := <-sigchnl
			if s == syscall.SIGINT {
				node.Shutdown()
				configs.P2PServerProto.Close()
				logging.LogInfo("Shutting down...")
				os.Exit(0)
			}
		}
	}()
}

func main() {
	cfgFilePath, _, logLevel := parseArgs()
	logging.InitLogging(logLevel)

	confs := sds.NewSdsConfig(cfgFilePath)

	node := sds.GetNodeInstance(confs)

	p2pServer := sds.InitNodeServer(node)
	go p2pServer.Serve(confs.P2PServerProto)

	node.StartSyncTask()

	logging.LogInfo("SniffDogSniff started press CTRL-C to stop")
	shutdownHook(confs, node)

	webServer := webui.InitSdsWebServer(node)
	webServer.ServeWebUi(confs.WebServiceBindAddress)

}
