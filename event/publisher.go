package event

import "github.com/byteintellect/go_commons"

type Publisher interface {
	Publish(event go_commons.Event)
	PublishAsync(event go_commons.Event)
	Flush()
	Close()
}
