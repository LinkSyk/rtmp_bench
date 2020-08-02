package util

type Messager interface {
	RecvMsg(fi *FlvInfo)
	Close()
}
