package background

import (
	_type "lifeboat/type"
	"lifeboat/utils"
	"strconv"
	"time"
)

type Healthcheck struct {
	interval time.Duration
	bgInfo   *BgInfo
	probe    IProbe
}

func NewHealthcheck(bgInfo *BgInfo, probe IProbe) *Healthcheck {
	rawInterval := utils.GetEnvOrDefault("HEALTHCHECK_INTERVAL", "3000")
	interval, _ := strconv.Atoi(rawInterval)
	return &Healthcheck{
		interval: time.Duration(interval) * time.Millisecond,
		bgInfo:   bgInfo,
		probe:    probe,
	}
}

func (h *Healthcheck) Start() {
	for true {
		h.healthcheck()
		time.Sleep(h.interval)
	}
}

func (h *Healthcheck) healthcheck() {
	result := h.probe.exec()
	if result {
		h.bgInfo.UpDownState = _type.UP
	} else {
		h.bgInfo.UpDownState = _type.DOWN
	}
}
