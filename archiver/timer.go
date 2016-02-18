package archiver

import (
	"time"
)

type ExponentialTimer struct {
	maximumTime int64
	currentTime int64
}

func NewExponentialTimer(max int64) *ExponentialTimer {
	return &ExponentialTimer{max, 1}
}

func (et *ExponentialTimer) Wait(showTime bool) {
	oldTime := et.currentTime
	if et.currentTime < et.maximumTime {
		tmp := oldTime << 1
		if tmp > et.maximumTime {
			tmp = et.maximumTime
		}
		et.currentTime = tmp
	}
	if showTime {
		log.Debugf("Waiting %v seconds for new connection", oldTime)
	}
	time.Sleep(time.Duration(oldTime) * time.Second)
}

func (et *ExponentialTimer) Reset() {
	et.currentTime = 1
}
