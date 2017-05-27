package raft

import (
	"context"
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
	"time"
)

type LeaderRole struct {
	dispatcher *Dispatcher
	replicas   map[int64]*Replica
}

func newLeaderRole(dispatcher *Dispatcher) *LeaderRole {
	return &LeaderRole{
		dispatcher: dispatcher,
		replicas:   make(map[int64]*Replica),
	}
}

const heartbeatTimeoutKey int = 174

func (r *LeaderRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	timeout := generateHeartbeetTimeout(state.timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	ctx = context.WithValue(ctx, heartbeatTimeoutKey, timeout)
	defer cancel()
	appendEntriesBuilder := state.appendEntriesRequestBuilder()
	for _, peer := range r.replicas {
		peer.nextIndex = state.log.Size()
		go peerThread(ctx, peer, appendEntriesBuilder)
	}
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := r.requestVoteResponse(state, requestVote.in)
			requestVote.send(response)
			if response.VoteGranted {
				return FollowerRoleHandle, state
			}
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, accepted := r.appendEntriesResponse(state, appendEntries.in)
			appendEntries.send(response)
			if accepted {
				return FollowerRoleHandle, state
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			executeCommand.sendError(fmt.Errorf("Not yest implemented"))
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}

func (r *LeaderRole) requestVoteResponse(state *State, in *pb.RequestVoteRequest) *pb.RequestVoteResponse {
	if in.Term <= state.currentTerm {
		return &pb.RequestVoteResponse{
			Term:        state.currentTerm,
			VoteGranted: false,
		}
	} else {
		return state.requestVoteResponse(in)
	}
}

func (r *LeaderRole) appendEntriesResponse(state *State, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, bool) {
	if in.Term <= state.currentTerm {
		// TODO: something really wrong happened if in.Term == state.currentTerm
		return &pb.AppendEntriesResponse{
			Term:    state.currentTerm,
			Success: false,
		}, false
	} else {
		return state.appendEntriesResponse(in)
	}
}

func peerThread(ctx context.Context, peer *Replica, builder func(LogReader, int64) *pb.AppendEntriesRequest) {
	timeout := ctx.Value(heartbeatTimeoutKey).(time.Duration)
	fakeLog := NewLog()
	for {
		select {
		case <-time.After(timeout):
			request := builder(fakeLog, peer.nextIndex)
			peer.client.AppendEntries(ctx, request)
		case <-ctx.Done():
			return
		}
	}
}

func generateHeartbeetTimeout(timeout time.Duration) time.Duration {
	ns := timeout.Nanoseconds()*5 / 10
	return time.Nanosecond * time.Duration(ns)
}
