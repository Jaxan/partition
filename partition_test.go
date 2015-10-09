package partition

import (
	"gitlab.science.ru.nl/rick/fsm"
	"math/rand"
	"testing"
)

func TestInitial(t *testing.T) {
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
	p := New(n, 2, true, f0, f1)

	for e := 0; e < n; e++ {
		element := p.partition[p.indices[e]].e
		block := p.partition[p.indices[e]].b
		if element != e {
			t.Errorf("Partition is indexed incorrectly for %d (e %d, b %d).", e, element, block)
		}
	}

	// 5 and 7 should be in the same block
	e := 5
	o := 7
	element := p.partition[p.indices[e]]
	other := p.partition[p.indices[o]]
	if element.b != other.b {
		t.Errorf("Elements %d and %d should be in the same block.", e, o)
	}

	// 0 and 40 should be in the same block
	e = 0
	o = 40
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b != other.b {
		t.Errorf("Elements %d and %d should be in the same block.", e, o)
	}

	// 0 and 40 should be in the same block
	e = 50
	o = 60
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b != other.b {
		t.Errorf("Elements %d and %d should be in the same block.", e, o)
	}

	// 55 and 57 should be in the same block
	e = 55
	o = 57
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b != other.b {
		t.Errorf("Elements %d and %d should be in the same block.", e, o)
	}

	// 5 and 55 should be in different blocks
	e = 5
	o = 55
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b == other.b {
		t.Errorf("Elements %d and %d should be in the different blocks.", e, o)
	}

	// 0 and 80 should be in different blocks
	e = 5
	o = 80
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b == other.b {
		t.Errorf("Elements %d and %d should be in the different blocks.", e, o)
	}

	// 4 and 5 should be in different blocks
	e = 4
	o = 5
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b == other.b {
		t.Errorf("Elements %d and %d should be in the different blocks.", e, o)
	}

	// 5 and 55 should be in different blocks
	e = 70
	o = 71
	element = p.partition[p.indices[e]]
	other = p.partition[p.indices[o]]
	if element.b == other.b {
		t.Errorf("Elements %d and %d should be in the different blocks.", e, o)
	}

	// TODO: test witnesses and parents
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

	p := New(states, outputs, true, o0, o1)
	p.Refine(t0, t1)

	blocks := make(map[int][]int, 6)
	count := 0
	for _ = range p.blockIndices(0, 0) {
		count++
	}
	if count != 6 {
		t.Errorf("Not all blocks are singletons (%v).", blocks)
	}

	// TODO test witnesses and parents
}

func TestWorstCase(t *testing.T) {
	for n := 10; n <= 1000; n = n * 10 {
		f := func(i int) int {
			if i == n-1 {
				return 0
			}
			return i + 1
		}
		class := func(i int) int {
			if i == n-1 {
				return 0
			}
			return 1
		}
		p := New(n, 2, true, class)
		p.Refine(f)
		c := 0
		for _ = range p.blockIndices(0, 0) {
			c++
		}

		count := <-p.count
		if c != count {
			t.Errorf("Block count is incorrect: expected %d, got %d.", c, count)
		}

		if c != n {
			t.Errorf("Block count is incorrect: expected %d, got %d.", c, count)
		}
	}
}

func benchmarkWorstCase(n int, b *testing.B) {
	t := func(i int) int {
		if i == 0 {
			return n - 1
		}
		return i - 1
	}
	o := func(i int) int {
		if i == 0 {
			return 0
		}
		return 1
	}
	for m := 0; m < b.N; m++ {
		p := New(n, 2, true, o)
		p.Refine(t)
	}
}

func BenchmarkWorstCase10(b *testing.B)      { benchmarkWorstCase(10, b) }
func BenchmarkWorstCase20(b *testing.B)      { benchmarkWorstCase(20, b) }
func BenchmarkWorstCase30(b *testing.B)      { benchmarkWorstCase(30, b) }
func BenchmarkWorstCase40(b *testing.B)      { benchmarkWorstCase(40, b) }
func BenchmarkWorstCase50(b *testing.B)      { benchmarkWorstCase(50, b) }
func BenchmarkWorstCase60(b *testing.B)      { benchmarkWorstCase(60, b) }
func BenchmarkWorstCase70(b *testing.B)      { benchmarkWorstCase(70, b) }
func BenchmarkWorstCase80(b *testing.B)      { benchmarkWorstCase(80, b) }
func BenchmarkWorstCase90(b *testing.B)      { benchmarkWorstCase(90, b) }
func BenchmarkWorstCase100(b *testing.B)     { benchmarkWorstCase(100, b) }
func BenchmarkWorstCase200(b *testing.B)     { benchmarkWorstCase(200, b) }
func BenchmarkWorstCase300(b *testing.B)     { benchmarkWorstCase(300, b) }
func BenchmarkWorstCase400(b *testing.B)     { benchmarkWorstCase(400, b) }
func BenchmarkWorstCase500(b *testing.B)     { benchmarkWorstCase(500, b) }
func BenchmarkWorstCase600(b *testing.B)     { benchmarkWorstCase(600, b) }
func BenchmarkWorstCase700(b *testing.B)     { benchmarkWorstCase(700, b) }
func BenchmarkWorstCase800(b *testing.B)     { benchmarkWorstCase(800, b) }
func BenchmarkWorstCase900(b *testing.B)     { benchmarkWorstCase(900, b) }
func BenchmarkWorstCase1000(b *testing.B)    { benchmarkWorstCase(1000, b) }
func BenchmarkWorstCase2000(b *testing.B)    { benchmarkWorstCase(2000, b) }
func BenchmarkWorstCase3000(b *testing.B)    { benchmarkWorstCase(3000, b) }
func BenchmarkWorstCase4000(b *testing.B)    { benchmarkWorstCase(4000, b) }
func BenchmarkWorstCase5000(b *testing.B)    { benchmarkWorstCase(5000, b) }
func BenchmarkWorstCase6000(b *testing.B)    { benchmarkWorstCase(6000, b) }
func BenchmarkWorstCase7000(b *testing.B)    { benchmarkWorstCase(7000, b) }
func BenchmarkWorstCase8000(b *testing.B)    { benchmarkWorstCase(8000, b) }
func BenchmarkWorstCase9000(b *testing.B)    { benchmarkWorstCase(9000, b) }
func BenchmarkWorstCase10000(b *testing.B)   { benchmarkWorstCase(10000, b) }
func BenchmarkWorstCase20000(b *testing.B)   { benchmarkWorstCase(20000, b) }
func BenchmarkWorstCase30000(b *testing.B)   { benchmarkWorstCase(30000, b) }
func BenchmarkWorstCase40000(b *testing.B)   { benchmarkWorstCase(40000, b) }
func BenchmarkWorstCase50000(b *testing.B)   { benchmarkWorstCase(50000, b) }
func BenchmarkWorstCase60000(b *testing.B)   { benchmarkWorstCase(60000, b) }
func BenchmarkWorstCase70000(b *testing.B)   { benchmarkWorstCase(70000, b) }
func BenchmarkWorstCase80000(b *testing.B)   { benchmarkWorstCase(80000, b) }
func BenchmarkWorstCase90000(b *testing.B)   { benchmarkWorstCase(90000, b) }
func BenchmarkWorstCase100000(b *testing.B)  { benchmarkWorstCase(100000, b) }
func BenchmarkWorstCase200000(b *testing.B)  { benchmarkWorstCase(200000, b) }
func BenchmarkWorstCase300000(b *testing.B)  { benchmarkWorstCase(300000, b) }
func BenchmarkWorstCase400000(b *testing.B)  { benchmarkWorstCase(400000, b) }
func BenchmarkWorstCase500000(b *testing.B)  { benchmarkWorstCase(500000, b) }
func BenchmarkWorstCase600000(b *testing.B)  { benchmarkWorstCase(600000, b) }
func BenchmarkWorstCase700000(b *testing.B)  { benchmarkWorstCase(700000, b) }
func BenchmarkWorstCase800000(b *testing.B)  { benchmarkWorstCase(800000, b) }
func BenchmarkWorstCase900000(b *testing.B)  { benchmarkWorstCase(900000, b) }
func BenchmarkWorstCase1000000(b *testing.B) { benchmarkWorstCase(1000000, b) }

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

func benchmarkPartition(states, inputs, outputs int, b *testing.B) {
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
		p := New(states, outputs, true, o...)
		p.Refine(t...)
	}
}

const inputs int = 10
const outputs int = 10

func BenchmarkPartition10(b *testing.B)      { benchmarkPartition(10, inputs, outputs, b) }
func BenchmarkPartition20(b *testing.B)      { benchmarkPartition(20, inputs, outputs, b) }
func BenchmarkPartition30(b *testing.B)      { benchmarkPartition(30, inputs, outputs, b) }
func BenchmarkPartition40(b *testing.B)      { benchmarkPartition(40, inputs, outputs, b) }
func BenchmarkPartition50(b *testing.B)      { benchmarkPartition(50, inputs, outputs, b) }
func BenchmarkPartition60(b *testing.B)      { benchmarkPartition(60, inputs, outputs, b) }
func BenchmarkPartition70(b *testing.B)      { benchmarkPartition(70, inputs, outputs, b) }
func BenchmarkPartition80(b *testing.B)      { benchmarkPartition(80, inputs, outputs, b) }
func BenchmarkPartition90(b *testing.B)      { benchmarkPartition(90, inputs, outputs, b) }
func BenchmarkPartition100(b *testing.B)     { benchmarkPartition(100, inputs, outputs, b) }
func BenchmarkPartition200(b *testing.B)     { benchmarkPartition(200, inputs, outputs, b) }
func BenchmarkPartition300(b *testing.B)     { benchmarkPartition(300, inputs, outputs, b) }
func BenchmarkPartition400(b *testing.B)     { benchmarkPartition(400, inputs, outputs, b) }
func BenchmarkPartition500(b *testing.B)     { benchmarkPartition(500, inputs, outputs, b) }
func BenchmarkPartition600(b *testing.B)     { benchmarkPartition(600, inputs, outputs, b) }
func BenchmarkPartition700(b *testing.B)     { benchmarkPartition(700, inputs, outputs, b) }
func BenchmarkPartition800(b *testing.B)     { benchmarkPartition(800, inputs, outputs, b) }
func BenchmarkPartition900(b *testing.B)     { benchmarkPartition(900, inputs, outputs, b) }
func BenchmarkPartition1000(b *testing.B)    { benchmarkPartition(1000, inputs, outputs, b) }
func BenchmarkPartition2000(b *testing.B)    { benchmarkPartition(2000, inputs, outputs, b) }
func BenchmarkPartition3000(b *testing.B)    { benchmarkPartition(3000, inputs, outputs, b) }
func BenchmarkPartition4000(b *testing.B)    { benchmarkPartition(4000, inputs, outputs, b) }
func BenchmarkPartition5000(b *testing.B)    { benchmarkPartition(5000, inputs, outputs, b) }
func BenchmarkPartition6000(b *testing.B)    { benchmarkPartition(6000, inputs, outputs, b) }
func BenchmarkPartition7000(b *testing.B)    { benchmarkPartition(7000, inputs, outputs, b) }
func BenchmarkPartition8000(b *testing.B)    { benchmarkPartition(8000, inputs, outputs, b) }
func BenchmarkPartition9000(b *testing.B)    { benchmarkPartition(9000, inputs, outputs, b) }
func BenchmarkPartition10000(b *testing.B)   { benchmarkPartition(10000, inputs, outputs, b) }
func BenchmarkPartition20000(b *testing.B)   { benchmarkPartition(20000, inputs, outputs, b) }
func BenchmarkPartition30000(b *testing.B)   { benchmarkPartition(30000, inputs, outputs, b) }
func BenchmarkPartition40000(b *testing.B)   { benchmarkPartition(40000, inputs, outputs, b) }
func BenchmarkPartition50000(b *testing.B)   { benchmarkPartition(50000, inputs, outputs, b) }
func BenchmarkPartition60000(b *testing.B)   { benchmarkPartition(60000, inputs, outputs, b) }
func BenchmarkPartition70000(b *testing.B)   { benchmarkPartition(70000, inputs, outputs, b) }
func BenchmarkPartition80000(b *testing.B)   { benchmarkPartition(80000, inputs, outputs, b) }
func BenchmarkPartition90000(b *testing.B)   { benchmarkPartition(90000, inputs, outputs, b) }
func BenchmarkPartition100000(b *testing.B)  { benchmarkPartition(100000, inputs, outputs, b) }
func BenchmarkPartition200000(b *testing.B)  { benchmarkPartition(200000, inputs, outputs, b) }
func BenchmarkPartition300000(b *testing.B)  { benchmarkPartition(300000, inputs, outputs, b) }
func BenchmarkPartition400000(b *testing.B)  { benchmarkPartition(400000, inputs, outputs, b) }
func BenchmarkPartition500000(b *testing.B)  { benchmarkPartition(500000, inputs, outputs, b) }
func BenchmarkPartition600000(b *testing.B)  { benchmarkPartition(600000, inputs, outputs, b) }
func BenchmarkPartition700000(b *testing.B)  { benchmarkPartition(700000, inputs, outputs, b) }
func BenchmarkPartition800000(b *testing.B)  { benchmarkPartition(800000, inputs, outputs, b) }
func BenchmarkPartition900000(b *testing.B)  { benchmarkPartition(900000, inputs, outputs, b) }
func BenchmarkPartition1000000(b *testing.B) { benchmarkPartition(1000000, inputs, outputs, b) }
