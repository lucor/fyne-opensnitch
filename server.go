// This file defines the OpenSnitch UI Server as per gRPC proto specification
package main

import (
	"context"

	"github.com/evilsocket/opensnitch/daemon/ui/protocol"
	"google.golang.org/grpc"
)

// Server represents an OpenSnitch UI Server
type osServer struct {
	server      *grpc.Server
	DefaultRule *protocol.Rule
	osApp       *osApp
}

// Ping reply to an OpenSnitch daemon client ping request and send to statChan the
// data received.
func (s *osServer) Ping(ctx context.Context, pr *protocol.PingRequest) (*protocol.PingReply, error) {
	statChan <- pr.GetStats()
	return &protocol.PingReply{Id: pr.GetId()}, nil
}

// AskRule ask to the UI application for a Rule for the specified connection
func (s *osServer) AskRule(ctx context.Context, conn *protocol.Connection) (*protocol.Rule, error) {
	// TODO send a default rule on timeout
	r, _ := s.osApp.AskRule(conn)
	return r, nil
}
