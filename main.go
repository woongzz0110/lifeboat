package main

import (
	log "github.com/sirupsen/logrus"
	"lifeboat/background"
	"lifeboat/raft"
	"lifeboat/type"
	"lifeboat/utils"
	"math"
	"strconv"
	"strings"
	"sync"
)

func main() {
	rawLifeboatPort := utils.GetEnvOrDefault("LIFEBOAT_PORT", "8080")
	lifeboatPort, _ := strconv.ParseInt(rawLifeboatPort, 10, 32)

	rawLifeboatId := utils.GetEnvOrDefault("LIFEBOAT_ID", "0")
	lifeboatId, _ := strconv.ParseInt(rawLifeboatId, 10, 64)

	rawLifeboatServers := utils.GetEnvOrDefault("LIFEBOAT_SERVERS", "")
	lifeboatServers := strings.Split(rawLifeboatServers, ",")
	var meInfo *raft.NodeInfo
	partnerInfos := &sync.Map{}
	for _, lifeboatServer := range lifeboatServers {
		rawServerId := strings.Split(lifeboatServer, ":")[0]
		serverId, _ := strconv.ParseInt(rawServerId, 10, 64)
		server := strings.Split(lifeboatServer, ":")[1] + ":" + strings.Split(lifeboatServer, ":")[2]
		if serverId == lifeboatId {
			meInfo = &raft.NodeInfo{
				Server:      server,
				NodeId:      serverId,
				UpDownState: _type.UP,
			}
		} else {
			partnerInfo := &raft.NodeInfo{
				Server: server,
				NodeId: serverId,
			}
			partnerInfos.Store(serverId, partnerInfo)
		}
	}
	nodeSize := len(lifeboatServers)
	raftInfo := &raft.RaftInfo{
		Quorum:   int(math.Ceil(float64(nodeSize+1) / 2)),
		LeaderId: -1,
		Epoch:    -1,
		RaftStep: raft.TODO,
	}

	bgInfo := &background.BgInfo{
		UpDownState: _type.DOWN,
	}

	healthcheckMode := utils.GetEnvOrDefault("HEALTHCHECK_MODE", "HTTP")
	healthcheckHttpMethod := utils.GetEnvOrDefault("HEALTHCHECK_HTTP_METHOD", "GET")
	healthcheckHttpUrl := utils.GetEnvOrDefault("HEALTHCHECK_HTTP_URL", "http://localhost")
	var probe background.IProbe
	switch healthcheckMode {
	case "HTTP":
		probe = background.NewHttpProbe(healthcheckHttpMethod, healthcheckHttpUrl)
	}

	go background.NewHealthcheck(bgInfo, probe).Start()
	go raft.NewHeartbeat(raftInfo, meInfo, partnerInfos, bgInfo).Start()
	err := raft.NewServer(raftInfo, meInfo, partnerInfos).Start(lifeboatPort)
	log.Errorf("Api Server start error. %+v", err)
}
