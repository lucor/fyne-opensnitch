package main

import (
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

func main() {

	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	statChan = make(chan *protocol.Statistics)
	app := newApp(sigChan)

	go func() {
		//Starts the gRPC server
		lis, err := net.Listen("unix", uiSocket)
		if err != nil {
			log.Error("OpenSnitch gRPC server failed to listen on socket unix://%s: %v", uiSocket, err)
		}

		server = grpc.NewServer()
		protocol.RegisterUIServer(
			server,
			&osServer{osApp: app},
		)

		log.Info("OpenSnitch gRPC server listening on socket: unix://%s", uiSocket)
		if err := server.Serve(lis); err != nil {
			log.Fatal("failed to serve: %v", err)
		}
	}()

	go app.ShowAndRun()
	for {
		select {
		case st := <-statChan:
			app.RefreshStats(st)
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
