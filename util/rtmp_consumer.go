package util

import (
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type RtmpConsumer struct {
	outputs    []string
	pReader    PacketReader
	publishers []Messager
}

var DeadLine = time.Second * 5

func NewRtmpConsumer(outputs []string, pReader PacketReader) Consumer {
	return &RtmpConsumer{
		outputs: outputs,
		pReader: pReader,
	}
}

func (rc *RtmpConsumer) Run(bufSize uint32) error {
	if !rc.checkRtmpUrl() {
		return errors.New("check rtmp url fail")
	}
	size := len(rc.outputs)
	var wg sync.WaitGroup
	wg.Add(size)
	for _, url := range rc.outputs {
		p := NewPublisher(url, &wg, bufSize)
		if err := p.Run(); err != nil {
			log.WithError(err).Errorf("publisher run error")
			continue
		}
		rc.publishers = append(rc.publishers, p)
	}

	for {
		packet, err := rc.pReader.Read()
		// if err == EAGAIN {
		// 	continue
		// }

		if err == STREAMCLOSE {
			rc.CLoseAllPublisher()
			log.Infoln("consumer: stream close")
			break
		}

		if err != nil {
			rc.CLoseAllPublisher()
			break
		}

		rc.dispatchPacket(packet)
		// time.Sleep(20 * time.Millisecond)
	}

	wg.Wait()
	return nil
}

func (rc *RtmpConsumer) Terminate() {

}

func (rc *RtmpConsumer) checkRtmpUrl() bool {
	for _, url := range rc.outputs {
		if !strings.HasPrefix(url, "rtmp://") {
			return false
		}
	}
	return true
}

func (rc *RtmpConsumer) dispatchPacket(fi *FlvInfo) {
	for _, publisher := range rc.publishers {
		publisher.RecvMsg(fi)
	}
}

func (rc *RtmpConsumer) CLoseAllPublisher() {
	for _, publisher := range rc.publishers {
		publisher.Close()
	}
}
