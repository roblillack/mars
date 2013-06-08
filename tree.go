// pathtree implements a tree for fast path lookup.
//
// Restrictions
//
//   - Paths must be a '/'-separated list of strings, like a URL or Unix filesystem.
//   - All paths must begin with a '/'.
//   - Path elements may not contain a '/'.
//   - Path elements beginning with a ':' or '*' will be interpreted as wildcards.
//   - Trailing slashes are inconsequential.
//
// Wildcards
//
// Wildcards are named path elements that may match any strings in that
// location.  Two different kinds of wildcards are permitted:
//   - :var - names beginning with ':' will match any single path element.
//   - *var - names beginning with '*' will match one or more path elements.
//            (however, no path elements may come after a star wildcard)
//
// Algorithm
//
// Paths are mapped to the tree in the following way:
//   - Each '/' is a Node in the tree. The root node is the leading '/'.
//   - Each Node has edges to other nodes. The edges are named according to the
//     possible path elements at that depth in the path.
//   - Any Node may have an associated Leaf.  Leafs are terminals containing the
//     data associated with the path as traversed from the root to that Node.
//
// Edges are implemented as a map from the path element name to the next node in
// the path.
package pathtree

import (
	"errors"
	"strings"
)

type Node struct {
	edges    map[string]*Node // the various path elements leading out of this node.
	wildcard *Node            // if set, this node had a wildcard as its path element.
	leaf     *Leaf            // if set, this is a terminal node for this leaf.
	star     *Leaf            // if set, this path ends in a star.
	leafs    int              // counter for # leafs in the tree
}

type Leaf struct {
	Value     interface{} // the value associated with this node
	Wildcards []string    // the wildcard names, in order they appear in the path
	order     int         // the order this leaf was added
}

// New returns a new path tree.
func New() *Node {
	return &Node{edges: make(map[string]*Node)}
}

// Add a path and its associated value to the tree.
//   - key must begin with "/"
//   - key must not duplicate any existing key.
// Returns an error if those conditions do not hold.
func (n *Node) Add(key string, val interface{}) error {
	if key[0] != '/' {
		return errors.New("Path must begin with /")
	}
	n.leafs++
	return n.add(n.leafs, splitPath(key), nil, val)
}

func (n *Node) add(order int, elements, wildcards []string, val interface{}) error {
	if len(elements) == 0 {
		if n.leaf != nil {
			return errors.New("duplicate path")
		}
		n.leaf = &Leaf{
			order:     order,
			Value:     val,
			Wildcards: wildcards,
		}
		return nil
	}

	var el string
	el, elements = elements[0], elements[1:]

	// Handle wildcards.
	switch el[0] {
	case ':':
		if n.wildcard == nil {
			n.wildcard = New()
		}
		return n.wildcard.add(order, elements, append(wildcards, el[1:]), val)
	case '*':
		if n.star != nil {
			return errors.New("duplicate path")
		}
		n.star = &Leaf{
			order:     order,
			Value:     val,
			Wildcards: append(wildcards, el[1:]),
		}
		return nil
	}

	// It's a normal path element.
	e, ok := n.edges[el]
	if !ok {
		e = New()
		n.edges[el] = e
	}

	return e.add(order, elements, wildcards, val)
}

// Find a given path. Any wildcards traversed along the way are expanded and
// returned, along with the value.
func (n *Node) Find(key string) (leaf *Leaf, expansions []string) {
	if len(key) == 0 || key[0] != '/' {
		return nil, nil
	}

	return n.find(splitPath(key), nil)
}

func (n *Node) find(elements, exp []string) (leaf *Leaf, expansions []string) {
	if len(elements) == 0 {
		return n.leaf, exp
	}

	// If this node has a star, calculate the star expansions in advance.
	var starExpansion string
	if n.star != nil {
		starExpansion = strings.Join(elements, "/")
	}

	// Peel off the next element and look up the associated edge.
	var el string
	el, elements = elements[0], elements[1:]
	if nextNode, ok := n.edges[el]; ok {
		leaf, expansions = nextNode.find(elements, exp)
	}

	// Handle colon
	if n.wildcard != nil {
		wildcardLeaf, wildcardExpansions := n.wildcard.find(elements, append(exp, el))
		if wildcardLeaf != nil && (leaf == nil || leaf.order > wildcardLeaf.order) {
			leaf = wildcardLeaf
			expansions = wildcardExpansions
		}
	}

	// Handle star
	if n.star != nil && (leaf == nil || leaf.order > n.star.order) {
		leaf = n.star
		expansions = append(exp, starExpansion)
	}

	return
}

func splitPath(key string) []string {
	elements := strings.Split(key, "/")
	if elements[0] == "" {
		elements = elements[1:]
	}
	if elements[len(elements)-1] == "" {
		elements = elements[:len(elements)-1]
	}
	return elements
}
