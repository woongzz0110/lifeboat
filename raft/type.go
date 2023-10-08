package raft

import (
	_type "lifeboat/type"
	"time"
)

type NodeInfo struct {
	Server string
	NodeId int64
	_type.UpDownState
	LastCheckTime time.Time
}

type RaftInfo struct {
	Quorum   int
	LeaderId int64
	Epoch    int64
	RaftStep
}

// RaftStep FAILOVER_STEP
type RaftStep string

const (
	TODO RaftStep = "TODO"
)

type NodeResponse struct {
	NodeInfo
	RaftInfo
}

type ElectionRequest struct {
	CandidateId int64
	Epoch       int64
}

type ElectionResult string
type ElectionResponse struct {
	Result   ElectionResult
	LeaderId int64
	Epoch    int64
	Error    error
}

const (
	AGREE    ElectionResult = "AGREE"
	DISAGREE ElectionResult = "DISAGREE"
	ERROR    ElectionResult = "ERROR"
)
