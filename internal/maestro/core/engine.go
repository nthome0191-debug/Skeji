package core

import "fmt"

type Engine struct {
	flows map[string]Flow
}

func NewEngine(flows ...Flow) *Engine {
	m := map[string]Flow{}
	for _, f := range flows {
		m[f.Name()] = f
	}
	return &Engine{flows: m}
}

func (e *Engine) Run(flowName string, ctx *MaestroContext) error {
	f, exists := e.flows[flowName]
	if !exists {
		return fmt.Errorf("unsupported flow: %v", flowName)
	}
	for _, step := range f.Steps() {
		err := step.Execute(ctx)
		if err != nil {
			return fmt.Errorf("%s step failed, pipeline errored: %s", step.Name, err)
		}
	}
	return nil
}
