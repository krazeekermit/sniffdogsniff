package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
	"github.com/sniffdogsniff/webui"
)

const PID_FILE_NAME = "sds.pid"

func showHelp() {
	fmt.Println("SniffDogSniff v0.1")
	fmt.Println("\t-c or --config\tSpecify config file location")
	os.Exit(0)
}

func parseArgs() (string, bool, string) {
	cfgFilePath := "./config.ini"
	runAsDaemon := false
	logLevel := "info"

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
			logLevel = os.Args[i+1]
			i++
		}
	}
	return cfgFilePath, runAsDaemon, logLevel
}

func createPidFile(cfg sds.SdsConfig) (string, error) {
	pidFilePath := filepath.Join(cfg.WorkDirPath, PID_FILE_NAME)

	if util.FileExists(pidFilePath) {
		return "",
			errors.New("failed to create pid file. Another instance of SniffDogSniff is already running.")
	}

	fp, err := os.OpenFile(pidFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	fp.WriteString(fmt.Sprintf("%d", os.Getpid()))
	fp.Close()

	return pidFilePath, nil
}

func deletePidFile(pidFilePath string) {
	if util.FileExists(pidFilePath) {
		os.Remove(pidFilePath)
	}
}

func shutdownHook(configs sds.SdsConfig, node *sds.LocalNode, pidFilePath string) {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	go func() {
		for {
			s := <-sigchnl
			if s == syscall.SIGINT {
				node.Shutdown()
				configs.P2PServerProto.Close()
				deletePidFile(pidFilePath)
				logging.LogInfo("Shutting down...")
				os.Exit(0)
			}
		}
	}()
}

func main() {
	cfgFilePath, daemon, logLevelStr := parseArgs()
	logging.InitLogging(logging.StrToLogLevel(logLevelStr))

	cfg := sds.NewSdsConfig(cfgFilePath)

	isDaemon := os.Getenv("SDS_IS_DAEMON") == "true"
	if daemon && !isDaemon {
		dPid, err := syscall.ForkExec(os.Args[0], os.Args[1:],
			&syscall.ProcAttr{
				Env: []string{"SDS_IS_DAEMON=true"},
				Sys: &syscall.SysProcAttr{
					Setsid: true,
				},
				Files: []uintptr{0, 1, 2},
			})

		if err != nil {
			panic("failed to start process daemon")
		}
		logging.LogInfo("SniffDogSniff daemon successfully started on pid", dPid)
		os.Exit(0)
	}

	// if it is running as daemon we force log to file
	if isDaemon || cfg.LogToFile {
		logging.SetLoggingToFile(filepath.Join(cfg.WorkDirPath, cfg.LogFileName))
	}

	pidFilePath, err := createPidFile(cfg)
	if err != nil {
		logging.LogError(err.Error())
		os.Exit(1)
	}

	node := sds.NewNode(cfg)

	p2pServer := sds.NewNodeServer(node)
	p2pServer.Serve(cfg.P2PServerProto)

	node.SetNodeAddress(cfg.P2PServerProto.GetAddressString())
	node.StartSyncTask()

	logging.LogInfo("SniffDogSniff started press CTRL-C to stop")
	shutdownHook(cfg, node, pidFilePath)

	webServer := webui.InitSdsWebServer(node)
	webServer.ServeWebUi(cfg.WebServiceBindAddress)

}
