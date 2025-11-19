package core

import "skeji/pkg/client"

type MaestroContext struct {
	Input   map[string]any
	Process map[string]any
	Output  map[string]any
	Client  *client.Client
}

func NewMaestroContext(input map[string]any, client *client.Client) *MaestroContext {
	return &MaestroContext{
		Input:  input,
		Output: make(map[string]any),
		Client: client,
	}
}
