package kafka

import "context"

type MessageBroker interface {
	SendMessage(ctx context.Context, key, value []byte) error
	ReadMessage(ctx context.Context) (key, value []byte, err error)
	Close() error
}
