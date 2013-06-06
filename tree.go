// pathtree implements a tree for storing and looking up paths. It supports
// wildcard expansions.
//
// Errata
//
// Trailing slashes do not affect anything.
package pathtree

import (
	"errors"
	"strings"
)

type Node struct {
	edges    map[string]*Node // the various path elements leading out of this node.
	wildcard *Node            // if set, this node had a wildcard as its path element.
	leaf     *Leaf            // if set, this is a terminal node for this leaf.
	leafs    int              // counter for # leafs in the tree
}

type Leaf struct {
	Value     interface{} // the value associated with this node
	Wildcards []string    // the wildcard names, in order they appear in the path
	order     int         // the order this leaf was added
}

func New() *Node {
	return &Node{edges: make(map[string]*Node)}
}

// Add a path and its associated value to the Trie.
// key must begin with "/"
// Returns an error if:
// - the key is a duplicate
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

	if el[0] == ':' {
		if n.wildcard == nil {
			n.wildcard = New()
		}
		return n.wildcard.add(order, elements, append(wildcards, el[1:]), val)
	}

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

	leaf, exp := n.find(splitPath(key), nil)
	if leaf == nil {
		return nil, nil
	}
	return leaf, exp
}

func (n *Node) find(elements, exp []string) (leaf *Leaf, expansions []string) {
	if len(elements) == 0 {
		return n.leaf, exp
	}

	var el string
	el, elements = elements[0], elements[1:]
	if nextNode, ok := n.edges[el]; ok {
		leaf, expansions = nextNode.find(elements, exp)
	}
	if n.wildcard == nil {
		return
	}

	exp = append(exp, el)
	wildcardLeaf, wildcardExpansions := n.wildcard.find(elements, exp)
	if wildcardLeaf != nil && (leaf == nil || leaf.order > wildcardLeaf.order) {
		leaf = wildcardLeaf
		expansions = wildcardExpansions
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
