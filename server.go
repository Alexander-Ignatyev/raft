package raft

import (
	"context"
	"github.com/golang/glog"
)

type Server struct {
	dispatcher *Dispatcher
	ctx        context.Context
	cancel     context.CancelFunc
	roles      map[RoleHandle]Role
}

func newServer(ctx context.Context) *Server {
	s := &Server{dispatcher: newDispatcher(),
		roles: make(map[RoleHandle]Role),
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	return s
}

func (s *Server) getRole(handle RoleHandle) Role {
	if r, ok := s.roles[handle]; ok {
		return r
	} else {
		return exitRoleInstance
	}

}

func (s *Server) run(handle RoleHandle, state *State) {
	glog.Infof("Starting server")
	for ; handle != ExitRoleHandle; handle, state = s.getRole(handle).RunRole(s.ctx, state) {
		glog.Infof("Switching to new role: %v", handle)
	}
}

func (s *Server) Stop() {
	s.cancel()
}
