package util

import (

	// "rtmp_bench/lib/flv"

	"time"

	log "github.com/sirupsen/logrus"
	flv "github.com/zhangpeihao/goflv"
)

type RtmpProducer struct {
	input      string
	running    bool
	streamLoop bool
	pReader    PacketReader
}

func NewRtmpProducer(input string, streamLoop bool, bufSize uint32) Producer {
	return &RtmpProducer{
		input:      input,
		streamLoop: streamLoop,
		running:    true,
		pReader:    NewPacketReader(bufSize),
	}
}

func (rp *RtmpProducer) Run() (PacketReader, error) {
	var timeBase int64 = 0
	go func() {
		defer rp.pReader.Close()
		isStart := false
		log.Infoln("producer: streamLoop: ", rp.streamLoop)
		for rp.running && (!isStart || rp.streamLoop) {
			isStart = true
			var startTime uint64 = 0
			var endTime uint64 = 0
			log.Infoln("start publish av")
			if err := rp.PublishAV2(rp.input, &startTime, &endTime); err != nil {
				log.WithError(err).Errorf("publishav error, input: %s, timeBase: %d, startTime: %d, endTime: %d", rp.input, timeBase, startTime, endTime)
				return
			}

			timeBase += int64(endTime - startTime)
			log.Infof("startTime: %d, endTime: %d", startTime, endTime)
		}
		log.Infoln("producer: stream close")
	}()

	return rp.pReader, nil
}

func (rp *RtmpProducer) Close() {

}

func (rp *RtmpProducer) Terminate() {
	rp.running = false
}

// func (rp *RtmpProducer) isRtmpUrl(url string) bool {
// 	return strings.HasPrefix(url, "rtmp://")
// }

func (rp *RtmpProducer) PublishAV(input string, timeBase int64, startTime *int32, endTime *int32) error {
	// 判断是否是合法flv文件
	// if !strings.HasSuffix(rp.input, ".flv") {
	// 	return errors.New("not legeal flv url")
	// }

	// f, err := os.Open(rp.input)
	// if err != nil {
	// 	return errors.New("open flv file error")
	// }
	// defer f.Close()

	// de, err := flv.NewDemuxer(f)
	// if err != nil {
	// 	return errors.New("init flv demuxer error")
	// }

	// _, _, _, err = de.ReadHeader()
	// if err != nil {
	// 	return err
	// }

	// for rp.running {
	// 	tagType, tagSize, timeStamp, err := de.ReadTagHeader()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if *startTime < 0 {
	// 		*startTime = int32(timeStamp)
	// 	}
	// 	*endTime = int32(timeStamp)

	// 	msg, err := de.ReadTag(tagSize)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// 处理脚本数据
	// 	if tagType == flv.TagTypeScriptData {

	// 	}

	// 	dts := uint32(timeBase) + timeStamp
	// 	flvInfo := FlvInfo{
	// 		Data:      msg,
	// 		TagType:   tagType,
	// 		TagSize:   tagSize,
	// 		Timestamp: dts,
	// 	}
	// 	log.Infof("producer: read msg: %d", flvInfo.Timestamp)
	// 	if err := rp.pReader.WriteMsg(&flvInfo); err != nil {
	// 		return err
	// 	}
	// 	// time.Sleep(time.Millisecond * 50)
	// }
	return nil
}

func (rp *RtmpProducer) PublishAV2(input string, startTime *uint64, endTime *uint64) error {
	flvFile, err := flv.OpenFile(input)
	if err != nil {
		log.WithError(err).Errorln("open flv file error")
		return err
	}
	defer flvFile.Close()

	var lastSleep uint64 = 0
	for rp.running && !flvFile.IsFinished() {
		header, data, err := flvFile.ReadTag()
		if err != nil {
			log.WithError(err).Errorln("flv read tag error")
			return err
		}

		if *startTime == 0 {
			*startTime = uint64(header.Timestamp)
		}
		*endTime = uint64(header.Timestamp)

		dts := uint64(header.Timestamp) - *startTime
		flvInfo := &FlvInfo{
			Data:      data,
			TagType:   header.TagType,
			TagSize:   header.DataSize,
			Timestamp: dts,
		}
		// log.Infof("producer: read msg: %d", flvInfo.Timestamp)
		if err := rp.pReader.WriteMsg(flvInfo); err != nil {
			return err
		}

		if lastSleep <= 0 {
			lastSleep = flvInfo.Timestamp
		}

		if flvInfo.Timestamp-lastSleep >= 300 {
			time.Sleep(time.Millisecond * time.Duration(flvInfo.Timestamp-lastSleep))
			lastSleep = flvInfo.Timestamp
		}
	}
	return nil
}
