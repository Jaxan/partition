A partition P of N is a set of pairwise disjoint subsets of N, called blocks, whose union is N. If P and Q are partitions of N, Q is a refinement of P if every block of Q is contained in a block of P. As a special case, every partition is a refinement of itself. The problem we solve is that of finding the coarsest refinement under a set of functions for a given partition. Given apartition P of N and a set of functions F, where each function has the form f: N->N, we construct the coarsest refinement Q of P such that elements in the same block behave equivalent under F, i.e.: for each pair of blocks B, B' of Q and each function f either B ⊆ f(B') or B ∩ f(B') = ∅.

In addition, the partition constructed forms a splitting tree. A splitting tree for P is a rooted tree T with the following properties:
- Each node u in T is labeled by a subset of N
- The root is labeled by N
- The label of each inner node is partitioned by the labels of its children
- The leaves are labeled by the (current) blocks of P
- Each inner node u is associated with a minimal-length sequence of relation references that provide evidence that the values contained in different children of u are inequivalent

The sequence associated to inner nodes provides a minimal-length 'witness' for the inequivalence of different blocks.

In this implementation, all nodes of the splitting tree are represented as blocks.