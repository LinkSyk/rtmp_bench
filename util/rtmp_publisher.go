package util

import (
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yireyun/go-queue"
	"github.com/zhangpeihao/gortmp"
)

type Publisher struct {
	msgCh      chan *FlvInfo
	publishURL string
	wg         *sync.WaitGroup
	msgQueue   *queue.EsQueue
}

type Message struct {
	fi  *FlvInfo
	eof bool
}

func NewPublisher(publishURL string, wg *sync.WaitGroup, bufSize uint32) *Publisher {
	return &Publisher{
		msgCh:      make(chan *FlvInfo, bufSize),
		publishURL: publishURL,
		wg:         wg,
		msgQueue:   queue.NewQueue(bufSize),
	}
}

func (p *Publisher) Run() error {
	ch := make(chan gortmp.OutboundStream)
	handler := &PublishHandler{streamCh: ch, msgQueue: p.msgQueue, wg: p.wg}
	connURL, err := p.GetConnectURL()
	if err != nil {
		return errors.New("GetConnectURL")
	}

	obConn, err := gortmp.Dial(connURL, handler, 100)
	if err != nil {
		log.WithError(err).Errorf("rtmp Dial error, url: %s", connURL)
		return err
	}

	if err := obConn.Connect(); err != nil {
		log.WithError(err).Errorf("rtmp connect error, url: %s", connURL)
		return err
	}

	go func() {
		for {
			select {
			case stream := <-ch:
				stream.Attach(handler)
				streamName, err := p.GetStreamName()
				if err != nil {
					log.WithError(err).Errorf("GetStreamName erorr")
					return
				}
				if err := stream.Publish(streamName, "live"); err != nil {
					log.WithError(err).Errorf("publish stream error")
					return
				}
			default:
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	return nil
}

func (p *Publisher) GetStreamName() (string, error) {
	ss := strings.Split(p.publishURL, "/")
	if len(ss) == 0 {
		return "", errors.New("invalid rtmp url")
	}
	return ss[len(ss)-1], nil
}

func (p *Publisher) GetConnectURL() (string, error) {
	ss := strings.Split(p.publishURL, "/")
	if len(ss) == 0 {
		return "", errors.New("invalid rtmp url")
	}
	return strings.Join(ss[0:len(ss)-1], "/"), nil
}

func (p *Publisher) RecvMsg(fi *FlvInfo) {
	msg := &Message{
		fi:  fi,
		eof: false,
	}

	p.msgQueue.Put(msg)
}

func (p *Publisher) Close() {
	msg := &Message{
		fi:  nil,
		eof: true,
	}

	p.msgQueue.Put(msg)
}

type PublishHandler struct {
	streamCh chan gortmp.OutboundStream
	msgQueue *queue.EsQueue
	wg       *sync.WaitGroup
}

func (ph *PublishHandler) OnReceived(conn gortmp.Conn, msg *gortmp.Message) {
}

func (ph *PublishHandler) OnReceivedRtmpCommand(conn gortmp.Conn, command *gortmp.Command) {
}

func (ph *PublishHandler) OnClosed(conn gortmp.Conn) {
}

func (ph *PublishHandler) OnStatus(obConn gortmp.OutboundConn) {
}

func (ph *PublishHandler) OnStreamCreated(obConn gortmp.OutboundConn, stream gortmp.OutboundStream) {
	ph.streamCh <- stream
}

func (handler *PublishHandler) OnPlayStart(stream gortmp.OutboundStream) {

}
func (handler *PublishHandler) OnPublishStart(stream gortmp.OutboundStream) {
	// Set chunk buffer size
	go handler.PublishData(stream)
}

func (handler *PublishHandler) PublishData(stream gortmp.OutboundStream) {
	// 从msgChan 中接受消息
	for {
		m, ok, _ := handler.msgQueue.Get()
		if !ok {
			time.Sleep(time.Millisecond * 20)
			continue
		}

		msg := m.(*Message)
		if msg.eof {
			log.Infoln("publisher: stream close")
			stream.Close()
			handler.wg.Done()
			return
		}

		if err := stream.PublishData(uint8(msg.fi.TagType), msg.fi.Data, uint32(msg.fi.Timestamp)); err != nil {
			return
		}
	}
}
