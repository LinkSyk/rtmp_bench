package util

type Producer interface {
	Run() (PacketReader, error)
	Terminate()
}
