package partition

import (
	"gitlab.science.ru.nl/rick/fsm"
	//"math/rand"
	"testing"
)

// A simple function for checking equality of int slices (such as witnesses).
func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestNew(t *testing.T) {
	n := 100
	f0 := func(i int) int {
		if i < 100/2 {
			return 0
		}
		return 1
	}
	f1 := func(i int) int {
		return i % 2
	}
	p := New(n, 2, f0, f1)

	for val := 0; val < n; val++ {
		for other := 0; other < n; other++ {
			witness := p.Witness(val, other)
			if f0(val) != f0(other) {
				if !equal(witness, []int{0}) {
					t.Errorf("Incorrect witness for %d and %d: %v.", val, other, witness)
				}
			} else if f1(val) != f1(other) {
				if !equal(witness, []int{1}) {
					t.Errorf("Incorrect witness for %d and %d: %v.", val, other, witness)
				}
			} else if !equal(witness, nil) {
				t.Errorf("Incorrect witness for %d and %d: %v.", val, other, witness)
			}
		}
	}
}

func TestRefine(t *testing.T) {
	states, inputs, outputs := 6, 2, 2
	transitions := []struct{ from, input, output, to int }{
		{0, 0, 0, 1},
		{0, 1, 0, 0},
		{1, 0, 1, 2},
		{1, 1, 0, 0},
		{2, 0, 0, 3},
		{2, 1, 0, 3},
		{3, 0, 1, 4},
		{3, 1, 0, 4},
		{4, 0, 0, 5},
		{4, 1, 1, 5},
		{5, 0, 1, 0},
		{5, 1, 0, 0},
	}
	m := fsm.New(states, inputs, outputs)

	for _, t := range transitions {
		m.SetTransition(t.from, t.input, t.output, t.to)
	}

	o0, _ := m.OutputFunction(0)
	o1, _ := m.OutputFunction(1)
	t0, _ := m.TransitionFunction(0)
	t1, _ := m.TransitionFunction(1)

	p := New(states, outputs, o0, o1)
	p.Refine(t0, t1)

	if p.size != 6 {
		t.Errorf("Not all blocks are singletons (%v).", p.size)
	}

	tests := []struct {
		val, other int
		witness    []int
	}{
		{0, 0, nil},
		{1, 1, nil},
		{2, 2, nil},
		{3, 3, nil},
		{4, 4, nil},
		{5, 5, nil},
		{0, 1, []int{0}},
		{0, 2, []int{1, 0}},
		{0, 3, []int{0}},
		{0, 4, []int{1}},
		{0, 5, []int{0}},
		{1, 2, []int{0}},
		{1, 3, []int{0, 1}},
		{1, 4, []int{0}},
		{1, 5, []int{0, 1, 0}},
		{2, 3, []int{0}},
		{2, 4, []int{1}},
		{2, 5, []int{0}},
		{3, 4, []int{0}},
		{3, 5, []int{0, 1}},
		{4, 5, []int{0}},
	}

	for _, test := range tests {
		witness := p.Witness(test.val, test.other)
		if !equal(witness, test.witness) {
			t.Errorf("Incorrect witness for %d and %d: %v.", test.val, test.other, witness)
		}
	}
}

/*
func generateFSM(states, inputs, outputs int) *fsm.FSM {
	m := fsm.New(states, inputs, outputs)
	for from := 0; from < states; from++ {
		for input := 0; input < inputs; input++ {
			output := rand.Intn(outputs)
			to := rand.Intn(states)
			m.SetTransition(from, input, output, to)
		}
	}
	return m
}

func benchmarkPartition(method, states, inputs, outputs int, b *testing.B) {
	rand.Seed(int64(states))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		m := generateFSM(states, inputs, outputs)
		o := make([]func(int) int, inputs)
		t := make([]func(int) int, inputs)
		for input := 0; input < inputs; input++ {
			o[input], _ = m.OutputFunction(input)
			t[input], _ = m.TransitionFunction(input)
		}
		b.StartTimer()
		p := New(states, outputs, o...)
		p.Refine(method, t...)
	}
}

const inputs int = 10
const outputs int = 10

func BenchmarkPartition10(b *testing.B)      { benchmarkPartition(MOORE, 10, inputs, outputs, b) }
func BenchmarkPartition20(b *testing.B)      { benchmarkPartition(MOORE, 20, inputs, outputs, b) }
func BenchmarkPartition30(b *testing.B)      { benchmarkPartition(MOORE, 30, inputs, outputs, b) }
func BenchmarkPartition40(b *testing.B)      { benchmarkPartition(MOORE, 40, inputs, outputs, b) }
func BenchmarkPartition50(b *testing.B)      { benchmarkPartition(MOORE, 50, inputs, outputs, b) }
func BenchmarkPartition60(b *testing.B)      { benchmarkPartition(MOORE, 60, inputs, outputs, b) }
func BenchmarkPartition70(b *testing.B)      { benchmarkPartition(MOORE, 70, inputs, outputs, b) }
func BenchmarkPartition80(b *testing.B)      { benchmarkPartition(MOORE, 80, inputs, outputs, b) }
func BenchmarkPartition90(b *testing.B)      { benchmarkPartition(MOORE, 90, inputs, outputs, b) }
func BenchmarkPartition100(b *testing.B)     { benchmarkPartition(MOORE, 100, inputs, outputs, b) }
func BenchmarkPartition200(b *testing.B)     { benchmarkPartition(MOORE, 200, inputs, outputs, b) }
func BenchmarkPartition300(b *testing.B)     { benchmarkPartition(MOORE, 300, inputs, outputs, b) }
func BenchmarkPartition400(b *testing.B)     { benchmarkPartition(MOORE, 400, inputs, outputs, b) }
func BenchmarkPartition500(b *testing.B)     { benchmarkPartition(MOORE, 500, inputs, outputs, b) }
func BenchmarkPartition600(b *testing.B)     { benchmarkPartition(MOORE, 600, inputs, outputs, b) }
func BenchmarkPartition700(b *testing.B)     { benchmarkPartition(MOORE, 700, inputs, outputs, b) }
func BenchmarkPartition800(b *testing.B)     { benchmarkPartition(MOORE, 800, inputs, outputs, b) }
func BenchmarkPartition900(b *testing.B)     { benchmarkPartition(MOORE, 900, inputs, outputs, b) }
func BenchmarkPartition1000(b *testing.B)    { benchmarkPartition(MOORE, 1000, inputs, outputs, b) }
func BenchmarkPartition2000(b *testing.B)    { benchmarkPartition(MOORE, 2000, inputs, outputs, b) }
func BenchmarkPartition3000(b *testing.B)    { benchmarkPartition(MOORE, 3000, inputs, outputs, b) }
func BenchmarkPartition4000(b *testing.B)    { benchmarkPartition(MOORE, 4000, inputs, outputs, b) }
func BenchmarkPartition5000(b *testing.B)    { benchmarkPartition(MOORE, 5000, inputs, outputs, b) }
func BenchmarkPartition6000(b *testing.B)    { benchmarkPartition(MOORE, 6000, inputs, outputs, b) }
func BenchmarkPartition7000(b *testing.B)    { benchmarkPartition(MOORE, 7000, inputs, outputs, b) }
func BenchmarkPartition8000(b *testing.B)    { benchmarkPartition(MOORE, 8000, inputs, outputs, b) }
func BenchmarkPartition9000(b *testing.B)    { benchmarkPartition(MOORE, 9000, inputs, outputs, b) }
func BenchmarkPartition10000(b *testing.B)   { benchmarkPartition(MOORE, 10000, inputs, outputs, b) }
func BenchmarkPartition20000(b *testing.B)   { benchmarkPartition(MOORE, 20000, inputs, outputs, b) }
func BenchmarkPartition30000(b *testing.B)   { benchmarkPartition(MOORE, 30000, inputs, outputs, b) }
func BenchmarkPartition40000(b *testing.B)   { benchmarkPartition(MOORE, 40000, inputs, outputs, b) }
func BenchmarkPartition50000(b *testing.B)   { benchmarkPartition(MOORE, 50000, inputs, outputs, b) }
func BenchmarkPartition60000(b *testing.B)   { benchmarkPartition(MOORE, 60000, inputs, outputs, b) }
func BenchmarkPartition70000(b *testing.B)   { benchmarkPartition(MOORE, 70000, inputs, outputs, b) }
func BenchmarkPartition80000(b *testing.B)   { benchmarkPartition(MOORE, 80000, inputs, outputs, b) }
func BenchmarkPartition90000(b *testing.B)   { benchmarkPartition(MOORE, 90000, inputs, outputs, b) }
func BenchmarkPartition100000(b *testing.B)  { benchmarkPartition(MOORE, 100000, inputs, outputs, b) }
func BenchmarkPartition200000(b *testing.B)  { benchmarkPartition(MOORE, 200000, inputs, outputs, b) }
func BenchmarkPartition300000(b *testing.B)  { benchmarkPartition(MOORE, 300000, inputs, outputs, b) }
func BenchmarkPartition400000(b *testing.B)  { benchmarkPartition(MOORE, 400000, inputs, outputs, b) }
func BenchmarkPartition500000(b *testing.B)  { benchmarkPartition(MOORE, 500000, inputs, outputs, b) }
func BenchmarkPartition600000(b *testing.B)  { benchmarkPartition(MOORE, 600000, inputs, outputs, b) }
func BenchmarkPartition700000(b *testing.B)  { benchmarkPartition(MOORE, 700000, inputs, outputs, b) }
func BenchmarkPartition800000(b *testing.B)  { benchmarkPartition(MOORE, 800000, inputs, outputs, b) }
func BenchmarkPartition900000(b *testing.B)  { benchmarkPartition(MOORE, 900000, inputs, outputs, b) }
func BenchmarkPartition1000000(b *testing.B) { benchmarkPartition(MOORE, 1000000, inputs, outputs, b) }
*/
