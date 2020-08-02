package util

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"net"
	"rtmp_bench/lib/rtmp"
	"strings"
	"time"

	"github.com/ossrs/go-oryx-lib/amf0"
	"github.com/prometheus/common/log"
)

func init() {
	rand.Seed(time.Now().Unix())
}

var timeout = time.Second * 5

type RtmpConnector struct {
	conn     net.Conn
	rtmpURL  string
	streamID string
}

func NewRtmpConnector(rtmpURL string) *RtmpConnector {
	return &RtmpConnector{
		rtmpURL: rtmpURL,
	}
}

func (rc *RtmpConnector) Connect() error {
	domain := rc.getDomain()
	if domain == "" {
		return errors.New("domain is empty")
	}

	// addrs, err := net.LookupHost(domain)
	// if err != nil {
	// 	return err
	// }

	// if len(addrs) == 0 {
	// 	return errors.New("domain don't resolve")
	// }

	conn, err := net.Dial("tcp", domain)
	if err != nil {
		return err
	}

	rc.conn = conn
	return nil
}

func (rc *RtmpConnector) HandShake() error {
	source := rand.NewSource(time.Now().Unix())
	r := rand.New(source)
	handShake := rtmp.NewHandshake(r)

	var c0c1 bytes.Buffer
	err := handShake.WriteC0S0(&c0c1)
	if err != nil {
		return err
	}

	err = handShake.WriteC1S1(&c0c1)
	if err != nil {
		return err
	}

	_ = rc.conn.SetDeadline(time.Now().Add(timeout))
	n, err := rc.conn.Write(c0c1.Bytes())
	if err != nil {
		return err
	}
	log.Infof("write(c0c1) %d bytes to connection", n)

	// client <- s0s1s2
	_ = rc.conn.SetDeadline(time.Now().Add(timeout))
	s0s1s2 := make([]byte, 1536*2+1)
	n, err = io.ReadFull(rc.conn, s0s1s2)
	if err != nil {
		return err
	}
	log.Infof("read(s0s1s2) %d bytes from connection\n", n)

	var c2 bytes.Buffer
	err = handShake.WriteC2S2(&c2, s0s1s2[1:1537])
	if err != nil {
		return err
	}

	_ = rc.conn.SetDeadline(time.Now().Add(timeout))
	n, err = rc.conn.Write(c2.Bytes())
	if err != nil {
		return err
	}
	log.Infof("write(c2) %d bytes to connection\n", n)

	return nil
}

func (rc *RtmpConnector) ConnectApp() error {
	ca := rtmp.NewConnectAppPacket()
	ca.CommandName = "connect"
	ca.TransactionID = 1
	ca.CommandObject = amf0.NewObject()
	ca.Args = amf0.NewObject()

	ca.Args.Set("app", amf0.NewString("live"))
	ca.Args.Set("type", amf0.NewString("nonprivate"))
	ca.Args.Set("flashVer", amf0.NewString("FMS.3.1"))
	ca.Args.Set("tcUrl", amf0.NewString("rtmp://172.27.93.168:1935"))

	var buf bytes.Buffer
	p := rtmp.NewProtocol(&buf)
	err := p.WritePacket(ca, 1)
	if err != nil {
		return err
	}

	log.Infof("connect info: %s", buf.String())
	_ = rc.conn.SetDeadline(time.Now().Add(timeout))
	n, err := rc.conn.Write(buf.Bytes())
	if err != nil {
		return err
	}
	log.Infof("write(connectApp req) %d bytes to connection", n)

	_ = rc.conn.SetDeadline(time.Now().Add(timeout))
	connResp := make([]byte, 65)
	n, err = io.ReadFull(rc.conn, connResp)
	if err != nil {
		return err
	}
	log.Infof("read(connectApp resp) %d from connection", n)
	return nil
}

func (rc *RtmpConnector) publishStream() error {
	return nil
}

func (rc *RtmpConnector) WritePacket2Rtmp(fi *FlvInfo) error {
	return nil
}

func (rc *RtmpConnector) getDomain() string {
	sets := strings.Split(rc.rtmpURL, "/")
	if len(sets) < 3 {
		return ""
	}
	return sets[2]
}

type handeShakePacket struct {
	buf []byte
}

func NewHandShakePacket() *handeShakePacket {
	return &handeShakePacket{
		buf: make([]byte, 0),
	}
}

func (hsp *handeShakePacket) Write(b []byte) (n int, err error) {
	allSize := len(hsp.buf) + len(b)
	newBuf := make([]byte, allSize)

	buf := bytes.NewBuffer(newBuf)
	buf.Write(hsp.buf)
	buf.Write(b)

	hsp.buf = buf.Bytes()
	return len(b), nil
}

func (hsp *handeShakePacket) getBuf() []byte {
	return hsp.buf
}
