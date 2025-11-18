package core

type Flow interface {
	Name() string
	Steps() []Step
}
