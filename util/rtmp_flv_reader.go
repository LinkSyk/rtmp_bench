package util

type FlvInfo struct {
	Data      []byte
	TagType   byte
	TagSize   uint32
	Timestamp uint64
}

type FlvPacketReader struct {
	msgCh   chan *FlvInfo
	bufSize uint32
}

func NewPacketReader(bufSize uint32) PacketReader {
	return &FlvPacketReader{
		msgCh:   make(chan *FlvInfo, bufSize),
		bufSize: bufSize,
	}
}

func (pr *FlvPacketReader) WriteMsg(msg interface{}) error {
	fInfo, ok := msg.(*FlvInfo)
	if !ok {
		panic("unknow struct")
	}

	pr.msgCh <- fInfo
	return nil
}

func (pr *FlvPacketReader) Read() (*FlvInfo, error) {
	if f, ok := <-pr.msgCh; ok {
		return f, nil
	}
	return nil, STREAMCLOSE
}

func (pr *FlvPacketReader) Close() {
	close(pr.msgCh)
}
