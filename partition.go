// Package partition can be used to construct the coarsest refinement of a partition P of a set N of
// integers [0, n) with respect to one or more functions of type N->N.
package partition

// A partition P of N is a set of pairwise disjoint subsets of N, called blocks, whose union is N.
// If P and Q are partitions of N, Q is a refinement of P if every block of Q is contained in a
// block of P. As a special case, every partition is a refinement of itself. The problem we solve is
// that of finding the coarsest refinement under a set of functions for a given partition. Given a
// partition P of N and a set of functions F, where each function has the form f: N->N, we construct
// the coarsest refinement Q of P such that elements in the same block behave equivalent under F,
// i.e.: for each pair of blocks B, B' of Q and each function f either B ⊆ f(B') or B ∩ f(B') = ∅.
//
// In addition, the partition constructed forms a splitting tree.
// A splitting tree for P is a rooted tree T with the following properties:
// - Each node u in T is labeled by a subset of N
// - The root is labeled by N
// - The label of each inner node is partitioned by the labels of its children
// - The leaves are labeled by the (current) blocks of P
// - Each inner node u is associated with a minimal-length sequence of relation references that
// 	 provide evidence that the values contained in different children of u are inequivalent
//
// The sequences associated to inner nodes provides a minimal-length 'witness' for the inequality of
// different blocks.
type (
	Partition struct {
		indices   []int       // a slice of indices to elements, indexed by integers [0,n).
		elements  []element   // a partition of the elements, and their blocks.
		splitters chan *block // a buffered channel of inner blocks that are 'splitgroups'.
		size      int         // the number of (leaf) blocks in the partition.
	}
	element struct {
		value int
		block *block
	}
	block struct {
		begin, end int    // interval of elements that belong to this block.
		level      int    // number of times the elements of this block have been refined.
		parent     *block // a pointer to the parent of this block
		borders    []int  // can be used to infer intervals of (direct) children.
		witness    []int  // sequence that distinguishes pairs for which this block is the lca.
	}
)

// New constructs an initial partition for integers in the set N=[0,n). Two integers x and y in N
// are in the same block if they belong to the same class for all provided functions. It is assumed
// that the range of the class functions is [0, max) (i.e. f(x) < max for all x in N).
func New(n int, max int, fs ...func(int) int) *Partition {
	// Initialize partition.
	p := new(Partition)

	b := &block{0, n, 0, nil, nil, nil}
	p.size++

	p.indices = make([]int, n)
	p.elements = make([]element, n)
	for i := 0; i < n; i++ {
		p.indices[i] = i
		p.elements[i] = element{i, b}
	}

	p.splitters = make(chan *block, n)

	for prefix, class := range fs {
		witness := []int{prefix}
		for b := range p.Blocks(0, n) {
			parent := p.split(b, max, class, witness)
			if parent != nil {
				p.splitters <- parent
			}
		}
	}
	return p
}

// Makes the partition stable with respect to functions fs that map elements from N to N.
// This implementation uses Hopcroft's 'process the smaller half' method.
// TODO: this method contains a bug because it does not pass the tests.
func (p *Partition) Refine(fs ...func(int) int) {
	n := len(p.elements)

	// Construct preimage for all functions
	preimages := make([]func(int) []int, len(fs))
	for i, f := range fs {
		preimages[i] = preimage(f, n)
	}

	// Refine until there are no groups of splitters left, or if all blocks are singletons.
done:
	for {
		select {
		case splitter := <-p.splitters:

			// Identify largest subblock of splitter.
			largest := 0
			delta := 0
			begin := splitter.begin
			for cls, border := range append(splitter.borders, splitter.end) {
				if border-begin > delta {
					delta = border - begin
					largest = cls
				}
				begin = border
			}

			for f := range fs {
				witness := append([]int{f}, splitter.witness...)

				// Mark the predecessors of all but the largest subblock of the splitter.
				marks := make(map[*block][][]int, p.size)
				count := make(map[*block]int, p.size)
				classes := make(map[*block]int, p.size)
				for cls := 0; cls < len(splitter.borders)+1; cls++ {
					if cls == largest {
						continue
					}
					begin := splitter.begin
					if cls != 0 {
						begin = splitter.borders[cls-1]
					}
					end := splitter.end
					if cls != len(splitter.borders) {
						end = splitter.borders[cls]
					}
					seen := make(map[*block]bool, p.size)
					for i := begin; i < end; i++ {
						suc := p.value(i)
						for _, val := range preimages[f](suc) {
							b := p.Block(suc)
							_, exists := marks[b]
							if !exists {
								marks[b] = make([][]int, len(splitter.borders)+1)
							}
							marks[b][cls] = append(marks[b][cls], val)
							count[b]++
							if !seen[b] {
								classes[b]++
								seen[b] = true
							}
						}
					}
				}

				// Move the marked values to subblocks.
				for b, refinement := range marks {
					if count[b] == b.end-b.begin && classes[b] == 1 {
						continue
					}

					// A split has been made, so make a parent.
					parent := &block{b.begin, b.end, b.level, b.parent, make([]int, 0), witness}
					b.parent = parent

					pos := b.end
					for cls := 0; cls < len(refinement); cls++ {
						if len(refinement[cls]) == 0 {
							continue
						}
						sb := &block{pos - len(refinement[cls]), pos, parent.level + 1, parent, nil, nil}
						p.size++
						parent.borders = append([]int{pos}, parent.borders...)

						// Decrease pos and swap the value at the current pos with val.
						for _, val := range refinement[cls] {
							pos--
							i := p.index(val)
							other := p.value(pos)
							p.elements[pos] = element{val, sb}
							p.indices[val] = pos
							p.elements[i] = element{other, b}
							p.indices[other] = i
						}
					}

					// Update the end position and the parent of the original block.
					b.end = pos
				}
				if p.size == n {
					break done
				}
			}
		default:
			break done
		}
	}
	close(p.splitters)
}

/*
// Makes the partition stable with respect to functions fs that map elements from N to N.
// This implementation iterates over all elements of all blocks for all splitters.
func (p *Partition) Refine(fs ...func(int) int) {
	n := len(p.elements)

	// Refine until there are no groups of splitters left, or if all blocks are singletons.
done:
	for {
		select {
		case splitter := <-p.splitters:
			for prefix, f := range fs {
				witness := append([]int{prefix}, splitter.witness...)
				for b := range p.Blocks(0, n) {

					// Figure out the range of the successors of elements in b.
					begin := n
					end := 0
					for i := b.begin; i < b.end; i++ {
						val := p.value(i)
						suc := f(val)
						j := p.index(suc)
						if j < begin {
							begin = j
						}
						if j > end {
							end = j
						}
					}

					// If all successors of elements in b are in the splitgroup, try to split.
					if begin >= splitter.begin && end <= splitter.end {

						// class returns the index of the splitter in which the successor of e is.
						class := func(val int) (cls int) {
							suc := f(val)
							i := p.index(suc)
							for _, border := range splitter.borders {
								if i < border {
									return
								}
								cls++
							}
							return
						}

						parent := p.split(b, len(splitter.borders)+1, class, witness)
						if parent != nil {
							p.splitters <- parent
						}

					}

					if p.size == n {
						break done
					}
				}
			}
		default:
			break done
		}
	}
	close(p.splitters)
}
*/

// split puts the elements in block b in different subblocks based on the class of their value. It
// is assumed that the range of the class function is [0, max). Returns the parent block if the
// block was split, or nil if it was not.
func (p *Partition) split(b *block, max int, class func(int) int, witness []int) (parent *block) {
	refinement := make([][]int, max)
	for i := b.begin; i < b.end; i++ {
		val := p.elements[i].value
		cls := class(val)
		refinement[cls] = append(refinement[cls], val)
	}

	if len(refinement[class(p.elements[b.begin].value)]) == b.end-b.begin {
		// All elements have the same class. No moves are needed.
		return
	}

	// A split has been made, so make a parent.
	parent = &block{b.begin, b.end, b.level, b.parent, make([]int, 0), witness}
	b.parent = parent

	// Construct subblocks and move elements to them.
	pos := b.end
	first := true
	for cls := 0; cls < max; cls++ {
		if len(refinement[cls]) == 0 {
			continue
		}
		sb := b
		if !first { // make a new block.
			sb = &block{pos - len(refinement[cls]), pos, parent.level + 1, parent, nil, nil}
			parent.borders = append([]int{pos}, parent.borders...)
			p.size++
		} else { // modify interval and level of b == sb.
			sb.begin = pos - len(refinement[cls])
			sb.level = parent.level + 1
		}
		first = false
		for _, val := range refinement[cls] { // move element to subblock.
			pos--
			p.elements[pos] = element{val, sb}
			p.indices[val] = pos
		}
	}
	return
}

// Blocks returns a read channel that contains pointers to blocks for the elements in the interval
// begin-end, such that the block of element[begin] is the first block on the channel, and the block
// of element[end] is the first block that is NOT in the channel.  It is safe to split the blocks
// that are read from the channel (i.e. the next block will not be a newly created subblock).
func (p *Partition) Blocks(begin, end int) <-chan *block {
	ch := make(chan *block)
	go func() {
		defer close(ch)
		n := len(p.elements)
		if end > n {
			end = n
		}
		for i := begin; i < end; {
			b := p.elements[i].block
			i = b.end
			ch <- b
		}
	}()
	return ch
}

// block returns the block for the provided value.
func (p *Partition) Block(val int) *block {
	if val >= len(p.elements) {
		return nil
	}
	i := p.indices[val]
	return p.elements[i].block
}

// Size returns the number of blocks in the partition.
func (p *Partition) Size() int {
	return p.size
}

// value returns the value that is on the provided index in the partition.
func (p *Partition) value(i int) int {
	return p.elements[i].value
}

// index returns the index in p.elements for the provided value.
func (p *Partition) index(val int) int {
	return p.indices[val]
}

// Witness returns a minimal-length sequence that distinguishes the provided pair of values, or nil
// if the values are in the same block. This is the sequence that is stored on the LCA of the
// values' elements.
func (p *Partition) Witness(val, other int) []int {
	lca := p.LCA(val, other)
	return lca.witness
}

// LCA returns the block that is the 'lowest common ancestor' of the provided values. This is the
// last block in which all of the values were present.
func (p *Partition) LCA(vals ...int) *block {
	n := len(p.elements)
	begin := n
	end := 0
	for _, val := range vals {
		if val >= n {
			continue
		}
		i := p.index(val)
		if begin > i {
			begin = i
		}
		if end < i {
			end = i
		}
	}
	if begin > end {
		return nil
	}
	return p.lca(p.elements[begin].block, p.elements[end].block)
}

// lca iteratively searches for the block that is the lowest common ancestor of the two provided
// blocks.
func (p *Partition) lca(b, o *block) *block {
	if b == o {
		return b
	}
	if b.level < o.level {
		return p.lca(b, o.parent)
	} else if b.level > o.level {
		return p.lca(b.parent, o)
	} // else b.level == o.level
	return p.lca(b.parent, o.parent)
}

// preimage returns the preimage function of f. This is a function that takes an element i in the
// range [0,n) and returns a slice of elements j for which f(j) = i.
func preimage(f func(int) int, n int) func(int) []int {
	p := make([][]int, n)
	for i := 0; i < n; i++ {
		j := f(i)
		p[j] = append(p[j], i)
	}
	return func(j int) []int {
		return p[j]
	}
}
