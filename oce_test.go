package partition

import (
	"fmt"
	"gitlab.science.ru.nl/rick/fsm"
	"testing"
)

const PATH string = "/Users/rick/Desktop/two-functions-hyp162.dot"

func BenchmarkOce(b *testing.B) {
	b.StopTimer()

	m := fsm.DotFile(PATH)
	states := m.States()
	inputs := m.Inputs()
	outputs := m.Outputs()
	o := make([]func(int) int, inputs)
	t := make([]func(int) int, inputs)
	for input := 0; input < inputs; input++ {
		o[input], _ = m.OutputFunction(input)
		t[input], _ = m.TransitionFunction(input)
	}
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		p := New(states, outputs, true, o...)
		p.Refine(t...)
		count := <-p.count
		fmt.Println(count)
		p.count <- count
		blocks := 0
		for _ = range p.blockIndices(0, 0) {
			blocks++
		}
		fmt.Println(blocks)
	}
}
