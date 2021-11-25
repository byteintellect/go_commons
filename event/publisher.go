package event

import (
	"github.com/byteintellect/go_commons/entity"
)

type Publisher interface {
	Publish(event entity.Event)
	PublishAsync(event entity.Event)
	Flush()
	Close()
}
