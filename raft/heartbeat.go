package raft

import (
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"lifeboat/background"
	_type "lifeboat/type"
	"lifeboat/utils"
	"strconv"
	"sync"
	"time"
)

type Heartbeat struct {
	interval     time.Duration
	apiClient    *resty.Client
	clients      *sync.Map
	raftInfo     *RaftInfo
	meInfo       *NodeInfo
	partnerInfos *sync.Map
	bgInfo       *background.BgInfo
}

func NewHeartbeat(raftInfo *RaftInfo, meInfo *NodeInfo, partnerInfos *sync.Map, bgInfo *background.BgInfo) *Heartbeat {
	clients := &sync.Map{}
	partnerInfos.Range(func(key, value any) bool {
		partnerInfo := value.(*NodeInfo)
		clients.Store(partnerInfo.NodeId, NewClient(partnerInfo.Server))
		return true
	})
	rawInterval := utils.GetEnvOrDefault("HEARTBEAT_INTERVAL", "3000")
	interval, _ := strconv.Atoi(rawInterval)
	return &Heartbeat{
		interval:     time.Duration(interval) * time.Millisecond,
		apiClient:    resty.New(),
		clients:      clients,
		raftInfo:     raftInfo,
		meInfo:       meInfo,
		partnerInfos: partnerInfos,
		bgInfo:       bgInfo,
	}
}

func (h *Heartbeat) Start() {
	for true {
		h.heartbeat()
		time.Sleep(h.interval)
	}
}

func (h *Heartbeat) heartbeat() {
	clusterLeaderId := h.raftInfo.LeaderId
	clusterEpoch := h.raftInfo.Epoch
	h.clients.Range(func(key, value any) bool {
		nodeId := key.(int64)
		var partnerInfo *NodeInfo
		if load, ok := h.partnerInfos.Load(nodeId); ok {
			partnerInfo = load.(*NodeInfo)
		}
		client := value.(*Client)
		nodeResponse, err := client.GetNode(nodeId)
		if err != nil {
			partnerInfo.UpDownState = _type.DOWN
			if nodeId == h.raftInfo.LeaderId {
				log.Warnf("Leader Node (%d) is down", h.raftInfo.LeaderId)
				if h.isAllAgreeLeaderDown() {
					h.pleaseVoteMeAsLeader()
				}
			}
			return true
		}
		nodeInfo := nodeResponse.NodeInfo
		nodeInfo.LastCheckTime = time.Now()
		if nodeResponse.UpDownState != partnerInfo.UpDownState {
			log.Warnf("Node (%d) state changed, %s -> %s", nodeId, partnerInfo.UpDownState, nodeResponse.UpDownState)
		}
		h.partnerInfos.Store(nodeId, &nodeInfo)
		if nodeResponse.Epoch >= clusterEpoch {
			clusterEpoch = nodeResponse.Epoch
			clusterLeaderId = nodeResponse.LeaderId
		}
		return true
	})

	if clusterEpoch >= h.raftInfo.Epoch {
		if h.raftInfo.LeaderId != clusterLeaderId {
			log.Infof("Leader changed, old leaderId/epoch: %d/%d, new leaderId/epoch: %d/%d", h.raftInfo.LeaderId, h.raftInfo.Epoch, clusterLeaderId, clusterEpoch)
			h.raftInfo.Epoch = clusterEpoch
			h.raftInfo.LeaderId = clusterLeaderId
		}
	}

	if h.raftInfo.LeaderId == -1 {
		if h.bgInfo.UpDownState == _type.UP {
			h.pleaseVoteMeAsLeader()
		}
	} else {
		log.Infof("Current leader node: %d, epoch: %d", h.raftInfo.LeaderId, h.raftInfo.Epoch)
	}
}

func (h *Heartbeat) isAllAgreeLeaderDown() bool {
	votes := 1
	h.clients.Range(func(key, value any) bool {
		client := value.(*Client)
		leaderInfo, err := client.GetNode(h.raftInfo.LeaderId)
		if err != nil {
			return false
		}
		if leaderInfo.UpDownState == _type.DOWN {
			votes++
		}
		return true
	})
	if votes >= h.raftInfo.Quorum {
		log.Infof("Many Nodes agree Leader is down. votes/quorum: %d/%d", votes, h.raftInfo.Quorum)
		return true
	} else {
		return false
	}
}

func (h *Heartbeat) pleaseVoteMeAsLeader() {
	log.Infof("Try to promote this node(%d) as leader", h.meInfo.NodeId)
	votes := 1
	newEpoch := h.raftInfo.Epoch + 1
	h.clients.Range(func(key, value any) bool {
		nodeId := key.(int64)
		client := value.(*Client)
		electionResponse, err := client.PostElection(ElectionRequest{
			CandidateId: h.meInfo.NodeId,
			Epoch:       newEpoch,
		})
		if err != nil {
			return false
		}
		if electionResponse.Result == AGREE {
			log.Debugf("Node (%d) voted me.", nodeId)
			votes++
		} else if electionResponse.Result == DISAGREE {
			if electionResponse.Epoch > h.raftInfo.Epoch {
				h.raftInfo.LeaderId = electionResponse.LeaderId
				h.raftInfo.Epoch = electionResponse.Epoch
			}
		}
		return true
	})
	if votes >= h.raftInfo.Quorum {
		log.Infof("Elected as the new leader. votes/quorum: %d/%d", votes, h.raftInfo.Quorum)
		h.raftInfo.LeaderId = h.meInfo.NodeId
		h.raftInfo.Epoch = newEpoch
	} else {
		log.Infof("Failed to win the election as the new leader. votes/quorum: %d/%d", votes, h.raftInfo.Quorum)
	}
}
