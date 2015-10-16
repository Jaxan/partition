// Package partition can be used to construct the coarsest refinement of a partition P of a set N of
// integers [0, n) with respect to one or more functions of type N->N.
package partition

import "sync"

// A partition P of N is a set of pairwise disjoint subsets of N, called blocks, whose union is N.
// If P and Q are partitions of N, Q is a refinement of P if every block of Q is contained in a
// block of P. As a special case, every partition is a refinement of itself. The problem we solve is
// that of finding the coarsest refinement under a set of functions for a given partition. Given a
// partition P of N and a set of functions F, where each function has the form f: N->N, we construct
// the coarsest refinement Q of P such that elements in the same block behave equivalent under F,
// i.e.: for each pair of blocks B, B' of Q either B ⊆ f(B') or B ∩ f(B') = ∅.
//
// In addition, the partition constructed forms a splitting tree.
// A splitting tree for P is a rooted tree T with the following properties:
// - Each node u in T is labeled by a subset of N
// - The root is labeled by N
// - The label of each inner node is partitioned by the labels of its children
// - The leaves are labeled by the (current) blocks of P
// - Each inner node u is associated with a minimal-length sequence of relation references that
// 	 provide evidence that the elements contained in different children of u are inequivalent
//
// The sequence associated to an inner node provides a minimal-length 'witness' for the inequivalence of
// different child blocks.
type (
	Partition struct {
		// indices is a slice of indices in the partition, indexed by the elements from U.
		indices []int
		// partition is a slice of elements e and their blocks b such that elements that belong to
		// the same block are adjacent.
		partition []struct{ e, b int }
		// blocks is a slice that contains a pool of available blocks. The slice's length and
		// capacity are initialized to 2n-1, because that is the upper bound on blocks (nodes)
		// required to construct a splitting tree for the partition. The blocks themselves implement
		// a linked list to find the indexes of current blocks in the partition, and a tree to
		// represent the splitting tree.
		blocks []Block
		// available is a buffered channel that contains indices of available blocks.
		available chan int
		// splitgroups is a buffered channel that contains splitgroups.
		splitgroups chan splitgroup
		// degree is the number of children a node in the tree can have. It is determined by the
		// output domain of the function that is used to construct the initial partition
		degree int
		// count is a channel with one integer in it which represents the number of blocks.
		// This value is stored in a channel so we can guarantee synchronised accesss.
		count chan int
	}
	Block struct {
		// begin is the index of the first element of this block. It is only set for parent blocks
		// end is one-past the index in partition of the last element of this block.
		end int
		// next and parent are indices of other blocks for implementing the linked list and tree.
		next, parent int
		// depth indicates the level of this block in the splitting tree (0 for root)
		depth int
		// witness is a minimal-length sequence of indices in relations that show that the elements
		// in the children of this block are inequivalent.
		witness []int
	}
)

// Splitters are implemented as splitgroups. A splitgroup is a set of splitters that have the same
// parent. A splitter is a range in the partition that corresponds to a node of the splitting tree.
type (
	splitgroup struct {
		splitters []splitter
		parent    int
	}
	splitter struct {
		begin, end int
	}
)

// New constructs an initial partition for elements in the set N=[0,n). Two elements x and y in N
// are in the same block if they belong to the same class for all provided class functions. It is
// assumed that the range of the class functions is [0, degree) (i.e. f(x) < degree for all x in N).
// The value of isWitness indicates if classes should be stored as witness in the splitting tree.
func New(n, degree int, isWitness bool, functions ...func(int) int) *Partition {
	p := new(Partition)
	p.indices = make([]int, n)
	p.partition = make([]struct{ e, b int }, n)
	for i := 0; i < n; i++ {
		p.indices[i] = i
		p.partition[i] = struct{ e, b int }{i, 0}
	}
	p.blocks = make([]Block, (2*n)-1) // the maximum number of nodes in a tree with n leaves is 2n-1
	p.blocks[0] = Block{n, 0, 0, 0, nil}
	p.available = make(chan int, (2*n)-1)
	for i := 1; i < (2*n)-1; i++ {
		p.available <- i
	}
	p.splitgroups = make(chan splitgroup, n)
	p.degree = degree
	p.count = make(chan int, 1)
	p.count <- 1

	for class, f := range functions {
		var wg sync.WaitGroup
		for b := range p.blockIndices(0, 0) {
			wg.Add(1)
			go func(b int) {
				defer wg.Done()
				next := p.blocks[b].next
				isSplit := p.split(b, degree, f)
				if isSplit {
					var witness []int
					if isWitness {
						witness = []int{class}
					}
					p.makeParent(b, next, witness)
				}
			}(b)
		}
		wg.Wait()
	}
	return p
}

// The commented out Refine function below marks elements that should be moved, instead of copying
// the splitters. This leads to a significantly worse performance (than the one that is in use, see
// below). This might be because there are some bugs in the code below.
/*
func (p *Partition) Refine(functions ...func(int) int) {
	n := len(p.partition)

	// Construct preimage sets for all functions.
	preimages := make([]func(int) []int, len(functions))
	var wg sync.WaitGroup
	for f := range functions {
		wg.Add(1)
		go func(f int) {
			defer wg.Done()
			preimages[f] = Preimage(functions[f], n)
		}(f)
	}
	wg.Wait()

	// Refine the partition until it is stable.
done:
	for {
		select {
		case sg := <-p.splitgroups:
			for f := range functions {
				// Construct the witness
				suffix := p.blocks[sg.parent].witness
				witness := append([]int{f}, suffix...)

				// First mark the elements that should be moved because of this splitgroup.
				type mark struct{ e, cls int }
				marked := make(chan mark, n)
				count := make(map[int]int, n)
				degree := make(map[int]int, p.degree)
				for cls, sp := range sg.splitters {
					seen := make(map[int]bool, n)
					for _, successor := range p.partition[sp.begin:sp.end] {
						for _, e := range preimages[f](successor.e) {
							marked <- mark{e, cls}
							b := p.partition[p.indices[e]].b
							count[b]++
							if !seen[b] {
								degree[b]++
								seen[b] = true
							}
						}
					}
				}
				wg.Wait()

				// Create workers that will move the marked elements to subblocks.
				work := make(map[int]chan mark, len(count))
				for b := range count {
					if degree[b] == 1 && count[b] == p.Len(b) { // we don't have to split this block
						continue
					}
					work[b] = make(chan mark) //TODO buffered?
					wg.Add(1)
					go func(b int) {
						defer wg.Done()
						next := p.blocks[b].next
						subblocks := make(map[int]int, p.degree)
						last := b
						for m := range work[b] {
							e := m.e
							i := p.indices[e]
							cls := m.cls
							sb, exists := subblocks[cls]
							if !exists {
								sb = <-p.available
								subblocks[cls] = sb
								p.insertAfter(last, sb)
								last = sb
							}
							p.move(i, sb)
						}
						p.makeParent(b, next, witness)
					}(b)
				}

				// Now move marked elements if a worker exists for their block.
			marks:
				for {
					select {
					case m := <-marked:
						b := p.partition[p.indices[m.e]].b
						ch, exists := work[b]
						if exists {
							ch <- m
						}
					default:
						break marks
					}
				}

				// Close worker channels
				for _, ch := range work {
					close(ch)
				}
				wg.Wait() // and wait for them to be closed, so parents are created

				// We are done if each block is a singleton.
				c := <-p.count
				p.count <- c
				if c == n {
					break done
				}
			}
		default:
			break done
		}
	}
	close(p.available)
	close(p.splitgroups)
}
*/

// TODO description
func (p *Partition) Refine(functions ...func(int) int) {
	n := len(p.partition)

	// Construct preimage sets for all functions
	preimage := make([][][]int, len(functions))
	var wg sync.WaitGroup
	for f := range functions {
		wg.Add(1)
		go func(f int) {
			defer wg.Done()
			preimage[f] = make([][]int, n)
			for i := 0; i < n; i++ {
				j := functions[f](i)
				preimage[f][j] = append(preimage[f][j], i)
			}
		}(f)
	}
	wg.Wait()

	// Refine the partition until it is stable
done: // indentation of labels in go sucks
	for {
		select {
		case sg := <-p.splitgroups:
			suffix := p.blocks[sg.parent].witness
			// Construct splitters prior to splitting, because the order of a block might change.
			splitters := make([][]struct{ e, b int }, len(sg.splitters))
			// TODO
			for i, sp := range sg.splitters {
				splitters[i] = p.partition[sp.begin:sp.end]
			}
			for f := range functions {
				// TODO
				for _, splitter := range splitters {
					// subblock maps blocks and functions to the (target) subblock for an element
					subblock := make(map[int]func(int) int, n)
					// keep track of created siblings for each splitted block
					siblings := make(map[int][]int, n)
					seen := make(map[int]map[int]bool, n)
					for _, successor := range splitter {
						predecessors := preimage[f][successor.e]
						for _, e := range predecessors {
							predecessor := p.partition[p.indices[e]]
							if p.Len(predecessor.b) == 1 {
								continue
							}
							// Determine target subblock for this element, based on successor.b.
							if subblock[predecessor.b] == nil { // if no mapping exists, create one
								subblock[predecessor.b] = p.subblockFunction(predecessor.b)
								siblings[predecessor.b] = make([]int, p.degree)
								seen[predecessor.b] = make(map[int]bool, p.degree)
							}
							sb := subblock[predecessor.b](successor.b)
							p.move(p.indices[e], sb)
							if seen[predecessor.b][sb] == false {
								seen[predecessor.b][sb] = true
								siblings[predecessor.b] = append(siblings[predecessor.b], sb)
							}
						}
					}

					// Loop over the splitted blocks to see if we need to clean up.
					var wg sync.WaitGroup
					for b := range siblings {
						wg.Add(1)
						go func(b int) {
							defer wg.Done()
							first := siblings[b][0]
							// If the splitted block b is empty, we have to move states from the
							// eldest sibling back to b. This way, the 'linked list' in p.blocks
							// keeps working, and block 0 never gets released.
							if p.blocks[b].end == p.blocks[first].end {
								begin, end := p.Range(first)
								for i := begin; i < end; i++ {
									p.partition[i].b = b
								}
								p.blocks[b].next = p.blocks[first].next
								// TODO should we alter count here?
								p.available <- first
								siblings[b] = siblings[b][1:]
								if len(siblings[b]) == 0 {
									return
								}
							}
							// Now b and one or more of its siblings have states, we have to create
							// a parent for them.
							last := siblings[b][len(siblings[b])-1]
							next := p.blocks[last].next
							witness := append([]int{f}, suffix...)
							p.makeParent(b, next, witness)
						}(b)
					}
					wg.Wait()

					// We are done if each block is a singleton.
					count := <-p.count
					p.count <- count
					if count == n {
						break done
					}
				}
			}
		default:
			break done
		}
	}
	close(p.available)
	close(p.splitgroups)
}

// Range returns the index of the first and (one past) the last element in the block with index b.
// Only works as expected if b is the index of a block that is currently in the partition.
func (p *Partition) Range(b int) (begin, end int) {
	block := p.blocks[b]
	end = block.end
	if block.next != 0 {
		begin = p.blocks[block.next].end
	}
	return
}

// Len returns the number of elements in the block with index b.
// Only works as expected if b is the index of a block that is currently in the partition.
func (p *Partition) Len(b int) int {
	begin, end := p.Range(b)
	return end - begin
}

// Splits the elements in block b according to their class, and returns true if a split is made.
// Time complexity (amortised): O(degree + len(p.blocks[b]) * (1 + O(class)))
func (p *Partition) split(b int, degree int, class func(int) int) bool {
	begin, end := p.Range(b)
	// a slice of subblocks. Each element contains the indices that will go into the same subblock.
	subblocks := make([][]int, degree)
	for i := begin; i < end; i++ {
		e := p.partition[i].e
		cls := class(e)
                // This call to append should have amortised constant time:
		subblocks[cls] = append(subblocks[cls], e)
	}
        if len(subblocks[class(p.partition[begin].e)]) == end - begin {
                // All states are in the same subblock. No moves are needed.
                return false
        }
	// the index of the last subblock added to the partition
	last := b
	first := true
	pos := end
	for cls := 0 ; cls < degree ; cls++ {
		states := subblocks[cls]
		if len(states) > 0 {
			sb := b
			if ! first {
				sb = <-p.available
				p.insertAfter(last, sb)
                                p.blocks[sb].end = pos
				last = sb
			}
                        first = false
			for _, st := range states {
                                pos--
				p.partition[pos] = struct{ e, b int }{st, sb}
				p.indices[st] = pos
			}
		}
	}
        if first || last == b {
                panic("split: Assertion failed. Internal error.")
        }
        return true
}

// Recursively moves element with index i to the next block until it is in the target block.
// Panics if not any of the next blocks is indexed by b.
func (p *Partition) move(i, target int) {
	b := p.partition[i].b
	if b == target {
		return
	}
	n := p.blocks[b].next
	if n == 0 {
		panic("Attempt to move an element to an invalid block.")
	}
	pivot := p.blocks[n].end
	element := p.partition[i].e
	other := p.partition[pivot].e

	// Swap element and other, and increase next block's end (i.e. move element to next block).
	p.partition[i] = p.partition[pivot]
	p.partition[pivot] = struct{ e, b int }{element, n}
	p.indices[element] = pivot
	p.indices[other] = i

	p.blocks[n].end++

	// Recurse until b == target
	p.move(pivot, target)
}

// blockIndices returns a channel that contains all block indices from and to (not including) the
// provided index. If to is not a block index (or 0), it contains all blocks starting at from.
func (p *Partition) blockIndices(from, to int) <-chan int {
	ch := make(chan int, len(p.blocks))
	go func() {
		current := from
		for {
			// first store the index of the next block to avoid a data race
			b := p.blocks[current]
			next := b.next
			ch <- current
			if next == to || next == 0 {
				break
			}
			current = next
		}
		close(ch)
	}()
	return ch
}

// insertAfter constructs a new block, stores it at index i, and puts the index in between
// b and p.blocks[b].next.
func (p *Partition) insertAfter(b, i int) {
	end, _ := p.Range(b)
	n, depth := p.blocks[b].next, p.blocks[b].depth
	block := Block{end, n, 0, depth, nil}
	p.blocks[i] = block
	p.blocks[b].next = i
	count := <-p.count
	count++
	p.count <- count
}

// makeParent constructs a parent block, and sets it as parent for all blocks in between first
// and next, including first, but not including next. Moreover, it adds a splitgroup.
func (p *Partition) makeParent(first, next int, witness []int) {
	end, grandparent, depth := p.blocks[first].end, p.blocks[first].parent, p.blocks[first].depth
	index := <-p.available
	sg := splitgroup{make([]splitter, 0, p.degree), index}
	var largest int // index in sg.splitters of splitter we need to remove later
	var length int  // number of elements in the largest block so far
	for child := range p.blockIndices(first, next) {
		p.blocks[child].parent = index
		p.blocks[child].depth++
		cbegin, cend := p.Range(child)
		sg.splitters = append(sg.splitters, splitter{cbegin, cend})
		if length < p.Len(child) {
			largest = len(sg.splitters) - 1
			length = p.Len(child)
		}
	}
	parent := Block{end, next, grandparent, depth, witness}
	p.blocks[index] = parent
	sg.splitters = append(sg.splitters[:largest], sg.splitters[largest+1:]...)
	p.splitgroups <- sg
}

// subblockFunction constructs a map that returns the subblock for a given block and splitter block.
func (p *Partition) subblockFunction(b int) func(int) int {
	seen := make(map[int]bool, p.degree)
	m := make(map[int]int, p.degree)
	last := b
	return func(sb int) int {
		if seen[sb] == false { // construct new block after last
			i := <-p.available
			p.insertAfter(last, i)
			seen[sb] = true
			m[sb] = i
			last = i
		}
		return m[sb]
	}
}

// Preimage returns the preimage of f. This is a function that takes an element i in the range [0,n)
// and returns a slice of elements j for which f(j) = i.
func Preimage(f func(int) int, n int) func(int) []int {
	p := make([][]int, n)
	for i := 0; i < n; i++ {
		j := f(i)
		p[j] = append(p[j], i)
	}
	return func(j int) []int {
		return p[j]
	}
}
