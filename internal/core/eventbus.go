package core

type EventBus struct {
	subscribers []chan Event
}
