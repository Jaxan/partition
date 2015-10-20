package partition

import (
	"gitlab.science.ru.nl/rick/fsm"
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
		t.Errorf("Constant MOORE is incorrectly set (%v).", MOORE)
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
