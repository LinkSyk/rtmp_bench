package util

type Consumer interface {
	Run(bufSize uint32) error
	Terminate()
}
