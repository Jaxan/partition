package partition

import (
	"gitlab.science.ru.nl/rick/fsm"
	"math/rand"
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

	if HOPCROFT != 0 {
		t.Errorf("Constant HOPCROFT is incorrectly set (%v).", HOPCROFT)
	}
	if MOORE != 1 {
		t.Errorf("Constant HOPCROFT is incorrectly set (%v).", MOORE)
	}

	p := New(states, outputs, o0, o1)
	p.Refine(HOPCROFT, t0, t1)

	q := New(states, outputs, o0, o1)
	q.Refine(MOORE, t0, t1)

	if p.size != 6 {
		t.Errorf("Not all blocks are singletons in Hopcroft (%v).", p.size)
	}
	if q.size != 6 {
		t.Errorf("Not all blocks are singletons in Moore (%v).", p.size)
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
			t.Errorf("Incorrect witness for %d and %d in Hopcroft: %v.", test.val, test.other, witness)
		}
		witness = q.Witness(test.val, test.other)
		if !equal(witness, test.witness) {
			t.Errorf("Incorrect witness for %d and %d in Moore: %v.", test.val, test.other, witness)
		}
	}
}

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

const INPUTS int = 10
const OUTPUTS int = 10

func BenchmarkMoore10(b *testing.B)       { benchmarkPartition(MOORE, 10, INPUTS, OUTPUTS, b) }
func BenchmarkMoore20(b *testing.B)       { benchmarkPartition(MOORE, 20, INPUTS, OUTPUTS, b) }
func BenchmarkMoore30(b *testing.B)       { benchmarkPartition(MOORE, 30, INPUTS, OUTPUTS, b) }
func BenchmarkMoore40(b *testing.B)       { benchmarkPartition(MOORE, 40, INPUTS, OUTPUTS, b) }
func BenchmarkMoore50(b *testing.B)       { benchmarkPartition(MOORE, 50, INPUTS, OUTPUTS, b) }
func BenchmarkMoore60(b *testing.B)       { benchmarkPartition(MOORE, 60, INPUTS, OUTPUTS, b) }
func BenchmarkMoore70(b *testing.B)       { benchmarkPartition(MOORE, 70, INPUTS, OUTPUTS, b) }
func BenchmarkMoore80(b *testing.B)       { benchmarkPartition(MOORE, 80, INPUTS, OUTPUTS, b) }
func BenchmarkMoore90(b *testing.B)       { benchmarkPartition(MOORE, 90, INPUTS, OUTPUTS, b) }
func BenchmarkMoore100(b *testing.B)      { benchmarkPartition(MOORE, 100, INPUTS, OUTPUTS, b) }
func BenchmarkMoore200(b *testing.B)      { benchmarkPartition(MOORE, 200, INPUTS, OUTPUTS, b) }
func BenchmarkMoore300(b *testing.B)      { benchmarkPartition(MOORE, 300, INPUTS, OUTPUTS, b) }
func BenchmarkMoore400(b *testing.B)      { benchmarkPartition(MOORE, 400, INPUTS, OUTPUTS, b) }
func BenchmarkMoore500(b *testing.B)      { benchmarkPartition(MOORE, 500, INPUTS, OUTPUTS, b) }
func BenchmarkMoore600(b *testing.B)      { benchmarkPartition(MOORE, 600, INPUTS, OUTPUTS, b) }
func BenchmarkMoore700(b *testing.B)      { benchmarkPartition(MOORE, 700, INPUTS, OUTPUTS, b) }
func BenchmarkMoore800(b *testing.B)      { benchmarkPartition(MOORE, 800, INPUTS, OUTPUTS, b) }
func BenchmarkMoore900(b *testing.B)      { benchmarkPartition(MOORE, 900, INPUTS, OUTPUTS, b) }
func BenchmarkMoore1000(b *testing.B)     { benchmarkPartition(MOORE, 1000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore2000(b *testing.B)     { benchmarkPartition(MOORE, 2000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore3000(b *testing.B)     { benchmarkPartition(MOORE, 3000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore4000(b *testing.B)     { benchmarkPartition(MOORE, 4000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore5000(b *testing.B)     { benchmarkPartition(MOORE, 5000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore6000(b *testing.B)     { benchmarkPartition(MOORE, 6000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore7000(b *testing.B)     { benchmarkPartition(MOORE, 7000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore8000(b *testing.B)     { benchmarkPartition(MOORE, 8000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore9000(b *testing.B)     { benchmarkPartition(MOORE, 9000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore10000(b *testing.B)    { benchmarkPartition(MOORE, 10000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore20000(b *testing.B)    { benchmarkPartition(MOORE, 20000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore30000(b *testing.B)    { benchmarkPartition(MOORE, 30000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore40000(b *testing.B)    { benchmarkPartition(MOORE, 40000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore50000(b *testing.B)    { benchmarkPartition(MOORE, 50000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore60000(b *testing.B)    { benchmarkPartition(MOORE, 60000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore70000(b *testing.B)    { benchmarkPartition(MOORE, 70000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore80000(b *testing.B)    { benchmarkPartition(MOORE, 80000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore90000(b *testing.B)    { benchmarkPartition(MOORE, 90000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore100000(b *testing.B)   { benchmarkPartition(MOORE, 100000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore200000(b *testing.B)   { benchmarkPartition(MOORE, 200000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore300000(b *testing.B)   { benchmarkPartition(MOORE, 300000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore400000(b *testing.B)   { benchmarkPartition(MOORE, 400000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore500000(b *testing.B)   { benchmarkPartition(MOORE, 500000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore600000(b *testing.B)   { benchmarkPartition(MOORE, 600000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore700000(b *testing.B)   { benchmarkPartition(MOORE, 700000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore800000(b *testing.B)   { benchmarkPartition(MOORE, 800000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore900000(b *testing.B)   { benchmarkPartition(MOORE, 900000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore1000000(b *testing.B)  { benchmarkPartition(MOORE, 1000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore2000000(b *testing.B)  { benchmarkPartition(MOORE, 2000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore3000000(b *testing.B)  { benchmarkPartition(MOORE, 3000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore4000000(b *testing.B)  { benchmarkPartition(MOORE, 4000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore5000000(b *testing.B)  { benchmarkPartition(MOORE, 5000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore6000000(b *testing.B)  { benchmarkPartition(MOORE, 6000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore7000000(b *testing.B)  { benchmarkPartition(MOORE, 7000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore8000000(b *testing.B)  { benchmarkPartition(MOORE, 8000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore9000000(b *testing.B)  { benchmarkPartition(MOORE, 9000000, INPUTS, OUTPUTS, b) }
func BenchmarkMoore10000000(b *testing.B) { benchmarkPartition(MOORE, 10000000, INPUTS, OUTPUTS, b) }

func BenchmarkHopcroft10(b *testing.B)      { benchmarkPartition(HOPCROFT, 10, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft20(b *testing.B)      { benchmarkPartition(HOPCROFT, 20, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft30(b *testing.B)      { benchmarkPartition(HOPCROFT, 30, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft40(b *testing.B)      { benchmarkPartition(HOPCROFT, 40, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft50(b *testing.B)      { benchmarkPartition(HOPCROFT, 50, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft60(b *testing.B)      { benchmarkPartition(HOPCROFT, 60, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft70(b *testing.B)      { benchmarkPartition(HOPCROFT, 70, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft80(b *testing.B)      { benchmarkPartition(HOPCROFT, 80, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft90(b *testing.B)      { benchmarkPartition(HOPCROFT, 90, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft100(b *testing.B)     { benchmarkPartition(HOPCROFT, 100, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft200(b *testing.B)     { benchmarkPartition(HOPCROFT, 200, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft300(b *testing.B)     { benchmarkPartition(HOPCROFT, 300, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft400(b *testing.B)     { benchmarkPartition(HOPCROFT, 400, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft500(b *testing.B)     { benchmarkPartition(HOPCROFT, 500, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft600(b *testing.B)     { benchmarkPartition(HOPCROFT, 600, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft700(b *testing.B)     { benchmarkPartition(HOPCROFT, 700, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft800(b *testing.B)     { benchmarkPartition(HOPCROFT, 800, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft900(b *testing.B)     { benchmarkPartition(HOPCROFT, 900, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft1000(b *testing.B)    { benchmarkPartition(HOPCROFT, 1000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft2000(b *testing.B)    { benchmarkPartition(HOPCROFT, 2000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft3000(b *testing.B)    { benchmarkPartition(HOPCROFT, 3000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft4000(b *testing.B)    { benchmarkPartition(HOPCROFT, 4000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft5000(b *testing.B)    { benchmarkPartition(HOPCROFT, 5000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft6000(b *testing.B)    { benchmarkPartition(HOPCROFT, 6000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft7000(b *testing.B)    { benchmarkPartition(HOPCROFT, 7000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft8000(b *testing.B)    { benchmarkPartition(HOPCROFT, 8000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft9000(b *testing.B)    { benchmarkPartition(HOPCROFT, 9000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft10000(b *testing.B)   { benchmarkPartition(HOPCROFT, 10000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft20000(b *testing.B)   { benchmarkPartition(HOPCROFT, 20000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft30000(b *testing.B)   { benchmarkPartition(HOPCROFT, 30000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft40000(b *testing.B)   { benchmarkPartition(HOPCROFT, 40000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft50000(b *testing.B)   { benchmarkPartition(HOPCROFT, 50000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft60000(b *testing.B)   { benchmarkPartition(HOPCROFT, 60000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft70000(b *testing.B)   { benchmarkPartition(HOPCROFT, 70000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft80000(b *testing.B)   { benchmarkPartition(HOPCROFT, 80000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft90000(b *testing.B)   { benchmarkPartition(HOPCROFT, 90000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft100000(b *testing.B)  { benchmarkPartition(HOPCROFT, 100000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft200000(b *testing.B)  { benchmarkPartition(HOPCROFT, 200000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft300000(b *testing.B)  { benchmarkPartition(HOPCROFT, 300000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft400000(b *testing.B)  { benchmarkPartition(HOPCROFT, 400000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft500000(b *testing.B)  { benchmarkPartition(HOPCROFT, 500000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft600000(b *testing.B)  { benchmarkPartition(HOPCROFT, 600000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft700000(b *testing.B)  { benchmarkPartition(HOPCROFT, 700000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft800000(b *testing.B)  { benchmarkPartition(HOPCROFT, 800000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft900000(b *testing.B)  { benchmarkPartition(HOPCROFT, 900000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft1000000(b *testing.B) { benchmarkPartition(HOPCROFT, 1000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft2000000(b *testing.B) { benchmarkPartition(HOPCROFT, 2000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft3000000(b *testing.B) { benchmarkPartition(HOPCROFT, 3000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft4000000(b *testing.B) { benchmarkPartition(HOPCROFT, 4000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft5000000(b *testing.B) { benchmarkPartition(HOPCROFT, 5000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft6000000(b *testing.B) { benchmarkPartition(HOPCROFT, 6000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft7000000(b *testing.B) { benchmarkPartition(HOPCROFT, 7000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft8000000(b *testing.B) { benchmarkPartition(HOPCROFT, 8000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft9000000(b *testing.B) { benchmarkPartition(HOPCROFT, 9000000, INPUTS, OUTPUTS, b) }
func BenchmarkHopcroft10000000(b *testing.B) {
	benchmarkPartition(HOPCROFT, 10000000, INPUTS, OUTPUTS, b)
}
