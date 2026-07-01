package minecraft

import "heimdall/internal/core"

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "minecraft"
}

func (p *Plugin) Start() error {
	return nil
}

func (p *Plugin) Poll() ([]core.Event, error) {
	return []core.Event{}, nil
}
