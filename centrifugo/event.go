package centrifugo

import "time"

type Event[T any] struct {
	Type string    `json:"type"`
	Ts   time.Time `json:"ts"`
	Body T         `json:"body"`
}
