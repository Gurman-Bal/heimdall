package core

type Plugin interface {
	Name() string
	Start() error
	Poll() ([]Event, error)
}
