package core

type Step struct {
	Name    string
	Execute func(ctx *MaestroContext) error
}

func NewStep(name string, execute func(ctx *MaestroContext) error) *Step {
	return &Step{
		Name:    name,
		Execute: execute,
	}
}
