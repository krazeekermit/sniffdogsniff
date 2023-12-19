package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/proxies/i2p"
	"github.com/sniffdogsniff/proxies/tor"
	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/webui"
)

const MAIN = "main"

const SDS_VER = "0.2_beta"

const PID_FILE_NAME = "sds.pid"

func showHelp() {
	fmt.Println("SniffDogSniff", SDS_VER)
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

func createPidFile(cfg core.SdsConfig) (string, error) {
	pidFilePath := filepath.Join(cfg.WorkDirPath, PID_FILE_NAME)

	if util.FileExists(pidFilePath) {
		return "",
			errors.New("failed to create pid file, another instance of SniffDogSniff is already running")
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

func shutdownHook(node *core.LocalNode, pidFilePath string, cfg core.SdsConfig,
	samSession *i2p.I2PSamSession, torControl *tor.TorControlSession) {
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	go func() {
		for {
			s := <-sigchnl
			if s == syscall.SIGINT {
				logging.Infof(MAIN, "Shutting down...")
				node.Shutdown()
				if cfg.P2pHiddenService == proxies.TOR_PROXY_TYPE {
					err := torControl.DeleteOnion()
					if err != nil {
						logging.Errorf(MAIN, err.Error())
					}
				} else if cfg.P2pHiddenService == proxies.I2P_PROXY_TYPE {
					err := samSession.CloseSession()
					if err != nil {
						logging.Errorf(MAIN, err.Error())
					}
				}
				deletePidFile(pidFilePath)
				os.Exit(0)
			}
		}
	}()
}

func main() {
	cfgFilePath, daemon, logLevelStr := parseArgs()
	logging.InitLogging(logging.StrToLogLevel(logLevelStr))

	cfg := core.NewSdsConfig(cfgFilePath)

	logging.Infof(MAIN, "SniffDogSniff %s running on %s", SDS_VER, runtime.GOOS)

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
		logging.Infof(MAIN, "SniffDogSniff daemon successfully started on pid %d", dPid)
		os.Exit(0)
	}

	// if it is running as daemon we force log to file
	if isDaemon || cfg.LogToFile {
		logging.SetLoggingToFile(filepath.Join(cfg.WorkDirPath, cfg.LogFileName))
	}

	pidFilePath, err := createPidFile(cfg)
	if err != nil {
		logging.Errorf(MAIN, err.Error())
		os.Exit(1)
	}

	node := core.NewLocalNode(cfg)

	var torSession *tor.TorControlSession = nil
	var i2pSamSession *i2p.I2PSamSession = nil

	p2pServer := core.NewNodeServer(node)
	if cfg.P2pHiddenService == proxies.NONE_PROXY_TYPE {
		p2pServer.ListenTCP(cfg.P2PServerBindAddress)
		node.SetNodeAddress(cfg.P2PServerBindAddress)
	} else if cfg.P2pHiddenService == proxies.TOR_PROXY_TYPE {
		torSession = tor.NewTorControlSession()
		onionAddr, err := torSession.CreateOnionService(cfg.TorContext, cfg.P2PBindPort, cfg.WorkDirPath)
		if err != nil {
			panic(err.Error())
		}
		p2pServer.ListenTCP(fmt.Sprintf("127.0.0.1:%d", cfg.P2PBindPort))
		node.SetNodeAddress(onionAddr)
	} else if cfg.P2pHiddenService == proxies.I2P_PROXY_TYPE {
		session, err := i2p.NewI2PSamSession(cfg.I2pContext, cfg.WorkDirPath)
		if err != nil {
			panic(err.Error())
		}
		i2pSamSession = session
		p2pServer.ListenI2P(i2pSamSession.Listener(), i2pSamSession.Base32Addr)
	}

	proxies.InitProxySettings(cfg.I2pContext, i2pSamSession, cfg.TorSocks5Address, cfg.ForceTor)

	node.StartNodesLookupTask()
	node.StartPublishTask()

	logging.Infof(MAIN, "SniffDogSniff started press CTRL-C to stop")
	shutdownHook(node, pidFilePath, cfg, i2pSamSession, torSession)

	webServer := webui.InitSdsWebServer(node)
	webServer.ServeWebUi(cfg.WebServiceBindAddress)

}
