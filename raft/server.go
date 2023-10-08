package raft

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
	"lifeboat/type"
	"strconv"
	"sync"
)

type Server struct {
	app          *fiber.App
	apiClient    *resty.Client
	RaftInfo     *RaftInfo
	MeInfos      *NodeInfo
	PartnerInfos *sync.Map
}

func NewServer(raftInfo *RaftInfo, meInfos *NodeInfo, partnerInfos *sync.Map) *Server {
	return &Server{
		app:          fiber.New(),
		apiClient:    resty.New(),
		RaftInfo:     raftInfo,
		MeInfos:      meInfos,
		PartnerInfos: partnerInfos,
	}
}

func (s *Server) Start(port int64) error {
	s.getNode()
	s.postElection()
	return s.app.Listen(":" + strconv.FormatInt(port, 10))
}

func (s *Server) getNode() {
	s.app.Get("/nodes/:nodeId", func(c *fiber.Ctx) error {
		var nodeResponse NodeResponse
		nodeId := c.Params("nodeId")
		nId, _ := strconv.ParseInt(nodeId, 10, 64)
		if nId == s.MeInfos.NodeId {
			nodeResponse.NodeInfo = *s.MeInfos
			nodeResponse.RaftInfo = *s.RaftInfo
		} else {
			value, _ := s.PartnerInfos.Load(nId)
			nodeInfo := value.(*NodeInfo)
			res, err := s.apiClient.R().Get(nodeInfo.Server + "/nodes/" + strconv.FormatInt(nodeInfo.NodeId, 10))
			if err != nil {
				nodeResponse.UpDownState = _type.DOWN
			} else {
				body := res.Body()
				_ = json.Unmarshal(body, &nodeResponse)
			}
		}
		return c.JSON(nodeResponse)
	})
}

func (s *Server) postElection() {
	s.app.Post("/election", func(c *fiber.Ctx) error {
		var electionRequest ElectionRequest
		var electionResponse ElectionResponse
		body := c.Body()
		err := json.Unmarshal(body, &electionRequest)
		if err != nil {
			log.Errorf("ElectionRequest parsing error. %+v", err)
			electionResponse = ElectionResponse{
				Result: ERROR,
				Error:  err,
			}
		}
		log.Infof("ElectionRequest received. %+v", electionRequest)
		if electionRequest.Epoch > s.RaftInfo.Epoch {
			s.RaftInfo.LeaderId = electionRequest.CandidateId
			s.RaftInfo.Epoch = electionRequest.Epoch
			value, _ := s.PartnerInfos.Load(electionRequest.CandidateId)
			newLeaderInfo := value.(*NodeInfo)
			newLeaderInfo.UpDownState = _type.UP
			s.RaftInfo.LeaderId = newLeaderInfo.NodeId
			electionResponse = ElectionResponse{
				Result:   AGREE,
				LeaderId: s.RaftInfo.LeaderId,
				Epoch:    s.RaftInfo.Epoch,
			}
		} else {
			electionResponse = ElectionResponse{
				Result:   DISAGREE,
				LeaderId: s.RaftInfo.LeaderId,
				Epoch:    s.RaftInfo.Epoch,
			}
		}
		body2, _ := json.Marshal(electionResponse)
		return c.Send(body2)
	})
}
