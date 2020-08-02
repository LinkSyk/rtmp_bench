package util

type PacketReader interface {
	WriteMsg(msg interface{}) error
	Read() (*FlvInfo, error)
	Close()
}
