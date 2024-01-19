package workflow

import "github.com/0xPolygon/beethoven/silencer"

type Workflow struct {
	silencer silencer.Silencer,
	sequencer sequencer.Sequencer,
	aggregator aggregator.Aggregator,
}

func New() Workflow {
	return Workflow{}
}

func (w *Workflow) Execute() {

}
