package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/evilsocket/opensnitch/daemon/log"
	"github.com/evilsocket/opensnitch/daemon/ui/protocol"
	"google.golang.org/grpc"
)

const (
	uiSocket = "/tmp/osui.sock"
)

var (
	sigChan  chan os.Signal
	statChan chan *protocol.Statistics
	server   *grpc.Server
)

// flag variables
var (
	configFile string
	debug      = false
)

func main() {

	flag.StringVar(&configFile, "config", "~/.opensnitch/ui-config.json", "UI configuration file")
	flag.BoolVar(&debug, "debug", debug, "Enable debug logs.")
	flag.Parse()

	if debug {
		log.MinLevel = log.DEBUG
	} else {
		log.MinLevel = log.INFO
	}

	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	cfg, err := loadConfigFromFile(configFile)
	if err != nil {
		log.Error("Error loading configuration file %s. %v", configFile, err)
	}

	statChan = make(chan *protocol.Statistics)
	osApp := newApp(sigChan, cfg)

	go func() {
		//Starts the gRPC server
		lis, err := net.Listen("unix", uiSocket)
		if err != nil {
			log.Error("OpenSnitch gRPC server failed to listen on socket unix://%s: %v", uiSocket, err)
		}

		server = grpc.NewServer()
		protocol.RegisterUIServer(
			server,
			&osServer{
				osApp: osApp,
			},
		)

		log.Info("OpenSnitch gRPC server listening on socket: unix://%s", uiSocket)
		if err := server.Serve(lis); err != nil {
			log.Fatal("failed to serve: %v", err)
		}
	}()

	go osApp.ShowAndRun()
	for {
		select {
		case st := <-statChan:
			osApp.RefreshStats(st)
		case sig := <-sigChan:
			log.Raw("\n")
			log.Important("Got signal: %v", sig)
			log.Info("Shutting down grpc server...")
			server.Stop()
			log.Info("All done, bye")
			os.Exit(0)
		}
	}
}
