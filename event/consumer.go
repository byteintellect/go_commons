package event

import "sync"

type Consumer interface {
	Close()
	Consume(wg *sync.WaitGroup)
}
