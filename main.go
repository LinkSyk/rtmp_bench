package main

import (
	"bufio"
	"flag"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"rtmp_bench/util"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func ReadRtmpFile(fileName string) ([]string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	var urls []string
	buf := bufio.NewReader(f)
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}
		urls = append(urls, string(line))
	}
	return urls, nil
}

func init() {
	go func() {
		http.ListenAndServe("127.0.0.1:8888", nil)
	}()
}

func main() {

	// parse command args
	input := flag.String("i", "/home/syk/data/video.flv", "must be flv file")
	loop := flag.Bool("stream_loop", false, "publish stream loopback")
	rtmpFile := flag.String("f", "", "contain of rtmp url")
	outPut := flag.String("r", "", "publish url")
	flag.Parse()

	rtmpURL := []string{}
	if *rtmpFile != "" {
		urls, err := ReadRtmpFile(*rtmpFile)
		if err != nil {
			log.WithError(err).Errorln("parse rtmpfile error")
			os.Exit(0)
		}
		rtmpURL = urls
	} else {
		if *outPut == "" {
			log.Errorln("publish url is empty")
			os.Exit(0)
		}
		rtmpURL = append(rtmpURL, *outPut)
	}

	p := util.NewRtmpProducer(*input, *loop, 100)
	reader, err := p.Run()
	if err != nil {
		log.WithError(err).Errorln("rtmp producer run error")
		return
	}

	consumers := util.NewRtmpConsumer(rtmpURL, reader)
	go consumers.Run(100)

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, syscall.SIGINT)
	<-signalCh
	log.Infoln("terminate publish")
	p.Terminate()
	time.Sleep(time.Second * 5)
}
