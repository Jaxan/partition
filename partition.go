// Package partition can be used to construct the coarsest refinement of a partition P of a set N of
// integers [0, n) with respect to one or more functions of type N->N.
package partition

import "sort" // for binary search Search(), used in Moore's algorithm

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
// - Each inner node u is associated with a minimal-length linked list of relation references that
// 	 provide evidence that the values contained in different children of u are inequivalent
//
// The sequence associated to inner nodes provides a minimal-length 'witness' for the inequivalence
// of different blocks.
//
// In this implementation, all nodes of the splitting tree are represented as blocks.
type (
	Partition struct {
		indices   []int     // a slice of indices to elements, indexed by integers [0,n).
		elements  []element // a partition of the elements, and their blocks.
		blocks    []block   // a slice of blocks/ nodes of the splitting tree, maximum length 2n-1.
		splitters chan int  // a buffered channel of inner blocks that are 'splitgroups'.
		size      int       // the number of (leaf) blocks in the partition.
		degree    int       // the maximum number of subblocks (children) for a block (inner node).
	}
	element struct {
		value int
		block int
	}
	block struct {
		begin, end int      // interval of elements that belong to this block.
		parent     int      // a pointer to the parent of this block.
		pivots     []int    // can be used to infer intervals of (direct) children.
		witness    *witness // distinguishes pairs for which this block is the lca.
		marks      [][]int  // the 'temporary children' for this block.
	}
	witness struct {
		prefix int      // the witness' prefix.
		suffix *witness // a pointer to the witness that contains the suffix (or nil).
	}
)

// New constructs an initial partition for integers in the set [0,n). Two integers x and y
// are in the same block if they belong to the same class for all provided functions. It is assumed
// that the range of the class functions is [0, degree) (i.e. f(x) < degree for all x in [0,n)).
func New(n int, degree int, fs ...func(int) int) *Partition {
	// Initialize partition.
	p := new(Partition)

	p.blocks = make([]block, 1, 2*n-1)
	p.blocks[0] = block{0, n, -1, nil, nil, make([][]int, degree)}
	p.size = 1
	p.degree = degree

	p.indices = make([]int, n)
	p.elements = make([]element, n)
	for i := 0; i < n; i++ {
		p.indices[i] = i
		p.elements[i] = element{i, 0}
	}

	p.splitters = make(chan int, n)

	for prefix, class := range fs {
		w := &witness{prefix, nil}
		for b := range p.Blocks(0, n) {
			parent := p.split(b, degree, class, w)
			if parent >= 0 {
				p.splitters <- parent
			}
		}
	}
	return p
}

const HOPCROFT int = 0
const MOORE int = 1

// Refine constructs the coarsest refinement of the current partition such that elements in the same
// block behave equivalently under the set of functions fs (that map elements from [0,n) to [0,n),
// i.e.: for each pair of blocks B, B' of Q and each function f either B ⊆ f(B') or B ∩ f(B') = ∅.
func (p *Partition) Refine(strategy int, fs ...func(int) int) {
	switch strategy {
	case HOPCROFT:
		p.refineHopcroft(fs...)
	case MOORE:
		p.refineMoore(fs...)
	default:
		panic("Undefined strategy.")
	}
}

// refineHopcroft is an implementation of Refine that uses Hopcroft's 'process the smaller half'
// strategy. This strategy has a theoretical time complexity of O(pn log n), where p == len(fs).
func (p *Partition) refineHopcroft(fs ...func(int) int) {
	n := len(p.elements)

	// Construct preimage for all functions
	preimages := make([]func(int) []int, len(fs))
	for prefix, f := range fs {
		preimages[prefix] = preimage(f, n)
	}

	// Refine until there are no groups of splitters left, or if all blocks are singletons.
done:
	for {
		select {
		case splitter := <-p.splitters:

			// Identify largest subblock of splitter.
			// The loop below does not check the last subblock of splitter;
			// therefore, we put it here.
			largest := len(p.blocks[splitter].pivots)
			delta := p.blocks[splitter].end - p.blocks[splitter].pivots[largest-1]
			begin := p.blocks[splitter].begin
			for cls, pivot := range p.blocks[splitter].pivots {
				if pivot-begin > delta {
					delta = pivot - begin
					largest = cls
				}
				begin = pivot
			}

			for prefix := range fs {
				w := &witness{prefix, p.blocks[splitter].witness}

				// Mark the predecessors of all but the largest subblock of the splitter.
				//marks := make([]map[int][]int, len(p.blocks[splitter].pivots)+1)
				// marks[class][block] is a list of values in block whose successors are in the class-th child of the splitter
				count := make([]int, len(p.blocks))
				// count[block] is the number of values in block that have been marked
				markblocks := make([]int, 0, len(p.blocks))
				for cls := 0; cls < len(p.blocks[splitter].pivots)+1; cls++ {
					if cls == largest {
						continue
					}
					begin := p.blocks[splitter].begin
					if cls != 0 {
						begin = p.blocks[splitter].pivots[cls-1]
					}
					end := p.blocks[splitter].end
					if cls != len(p.blocks[splitter].pivots) {
						end = p.blocks[splitter].pivots[cls]
					}
					for i := begin; i < end; i++ {
						for _, val := range preimages[prefix](p.value(i)) {
							b := p.Block(val)
							if p.blocks[b].end-p.blocks[b].begin == 1 {
								// singleton block cannot be split
								continue
							}
							if count[b] == 0 {
								markblocks = append(markblocks, b)
							}
							p.blocks[b].marks[cls] = append(p.blocks[b].marks[cls], val)
							count[b]++
						}
					}
				}

				// Move the marked values to subblocks.
				for _, b := range markblocks {
					var parent int
					pos := p.blocks[b].end - count[b]
					if pos == p.blocks[b].begin {
						// the implicit child is empty.
						parent = b
					} else {
						parent = len(p.blocks)
						p.blocks = append(p.blocks, p.blocks[b])
						p.blocks[parent].pivots = []int{pos}

						p.blocks[b].end = pos
						p.blocks[b].parent = parent
					}

					first := true
					for cls := 0; cls < len(p.blocks[splitter].pivots)+1; cls++ {
						if cls == largest || p.blocks[b].marks[cls] == nil || len(p.blocks[b].marks[cls]) == 0 {
							continue
						}
						if first {
							first = false
							if len(p.blocks[b].marks[cls]) == p.blocks[parent].end-p.blocks[parent].begin {
								// not a real split
								p.blocks[b].marks[cls] = p.blocks[b].marks[cls][:0]
								break
							}
							p.blocks[parent].witness = w
							p.splitters <- parent
							if pos > p.blocks[parent].begin {
								p.size++
							}
						} else {
							p.blocks[parent].pivots = append(p.blocks[parent].pivots, pos)
							p.size++
						}
						sb := len(p.blocks)
						p.blocks = append(p.blocks, block{pos, pos + len(p.blocks[b].marks[cls]), parent, nil, nil, make([][]int, p.degree)})

						// Swap the value at the current pos with val and increase pos
						for _, val := range p.blocks[b].marks[cls] {
							i := p.index(val)
							other := p.value(pos)
							p.elements[pos] = element{val, sb}
							p.indices[val] = pos
							p.elements[i].value = other
							p.indices[other] = i
							pos++
						}
						p.blocks[b].marks[cls] = p.blocks[b].marks[cls][:0]
					}
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

// refineMoore is an implementation of Refine that iterates over all elements of all blocks and
// splitters. This strategy was proposed by Moore. It has a theoretical time complexity of O(pn²).
func (p *Partition) refineMoore(fs ...func(int) int) {
	n := len(p.elements)

	// Refine until there are no groups of splitters left, or until all blocks are singletons.
done:
	for {
		select {
		case splitter := <-p.splitters:
			for prefix, f := range fs {
				w := &witness{prefix, p.blocks[splitter].witness}
				for b := range p.Blocks(0, n) {

					// Figure out the range of the successors of elements in b.
					begin := n
					end := 0
					for i := p.blocks[b].begin; i < p.blocks[b].end; i++ {
						j := p.index(f(p.value(i)))
						if j < begin {
							begin = j
						}
						if j > end {
							end = j
						}
					}

					// If all successors of elements in b are in the splitgroup, try to split.
					if begin >= p.blocks[splitter].begin && end <= p.blocks[splitter].end {

						// class returns the index of the splitter in which the successor of e is.
						class := func(val int) int {
							x := p.index(f(val))
							return sort.Search(len(p.blocks[splitter].pivots), func(i int) bool { return p.blocks[splitter].pivots[i] > x })
						}

						parent := p.split(b, len(p.blocks[splitter].pivots)+1, class, w)
						if parent >= 0 {
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

// split puts the elements in block b in different subblocks based on the class of their value. It
// is assumed that the range of the class function is [0, degree). Returns the parent block identifier
// if the block was split, or -1 if it was not.
func (p *Partition) split(b int, degree int, class func(int) int, w *witness) (parent int) {
	parent = -1
	refinement := make([][]int, degree)
	begin := p.blocks[b].begin
	end := p.blocks[b].end
	for i := begin; i < end; i++ {
		val := p.elements[i].value
		cls := class(val)
		refinement[cls] = append(refinement[cls], val)
	}

	if len(refinement[class(p.elements[begin].value)]) == end-begin {
		// All elements have the same class. No moves are needed.
		return
	}

	// A split has been made, so make a parent.
	parent = len(p.blocks)
	p.blocks = append(p.blocks, block{begin, end, p.blocks[b].parent, make([]int, 0), w, nil})
	p.blocks[b].parent = parent

	// Construct subblocks and move elements to them.
	pos := begin
	first := true
	for cls := 0; cls < degree; cls++ {
		if len(refinement[cls]) == 0 {
			continue
		}
		sb := b
		if !first { // make a new block.
			sb = len(p.blocks)
			p.blocks = append(p.blocks, block{pos, pos + len(refinement[cls]), parent, nil, nil, make([][]int, p.degree)})
			p.blocks[parent].pivots = append(p.blocks[parent].pivots, pos)
			p.size++
		} else { // modify interval b == sb.
			p.blocks[sb].end = pos + len(refinement[cls])
			first = false
		}
		for _, val := range refinement[cls] { // move element to subblock.
			p.elements[pos] = element{val, sb}
			p.indices[val] = pos
			pos++
		}
	}
	return
}

// Blocks returns a read channel that contains block identifiers for the elements in the interval
// begin-end, such that the block of element[begin] is the first block on the channel, and the block
// of element[end] is the first block that is not in the channel.  It is safe to split the blocks
// that are read from the channel (i.e. the next block will not be a newly created subblock).
func (p *Partition) Blocks(begin, end int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		n := len(p.elements)
		if end > n {
			end = n
		}
		for i := begin; i < end; {
			b := p.elements[i].block
			i = p.blocks[b].end
			// It is safe to split block b, because the end of b is stored beforehand.
			ch <- b
		}
	}()
	return ch
}

// block returns the block identifier for the provided value, or -1 if the value is out of range.
func (p *Partition) Block(val int) int {
	if val >= len(p.elements) || val < 0 {
		return -1
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

// Witness returns a minimal-length sequence of function indexes that distinguishes the provided
// pair of values, or nil if the values are in the same block. This is the sequence that is stored
// on the LCA of the values' elements.
// TODO: use unbuffered channel? -- Implement 'Sequence' package with compare utility.
func (p *Partition) Witness(val, other int) []int {
	lca := p.LCA(val, other)
	ret := make([]int, 0, len(p.blocks)) // TODO: keep track of height of tree?
	for w := p.blocks[lca].witness; w != nil; w = w.suffix {
		ret = append(ret, w.prefix)
	}
	return ret
}

// LCA returns the block that is the 'lowest common ancestor' of the provided values. This is the
// lowest block in which all of the values were present.
func (p *Partition) LCA(vals ...int) int {
	n := len(p.elements)
	begin := n
	end := 0
	for _, val := range vals {
		if val < 0 || val >= n {
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
		return -1
	}
	return p.lca(p.elements[begin].block, p.elements[end].block)
}

// lca iteratively searches for the block that is the lowest common ancestor of the two provided
// blocks b and o. It is assumed that p.blocks[b].begin < p.blocks[o].end upon calling.
func (p *Partition) lca(b, o int) int {
	for p.blocks[b].end <= p.blocks[o].begin {
		b = p.blocks[b].parent
	}
	return b
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
