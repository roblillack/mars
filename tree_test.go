package pathtree

import (
	"reflect"
	"testing"
)

func TestColon(t *testing.T) {
	n := New()

	n.Add("/:first/:second/", 1)
	n.Add("/:first", 2)
	n.Add("/", 3)

	found(t, n, "/", nil, 3)
	found(t, n, "/a", []string{"a"}, 2)
	found(t, n, "/a/", []string{"a"}, 2)
	found(t, n, "/a/b", []string{"a", "b"}, 1)
	found(t, n, "/a/b/", []string{"a", "b"}, 1)

	notfound(t, n, "/a/b/c")
}

func TestStar(t *testing.T) {
	n := New()

	n.Add("/first/second/*star", 1)
	n.Add("/:first/*star/", 2)
	n.Add("/*star", 3)
	n.Add("/", 4)

	found(t, n, "/", nil, 4)
	found(t, n, "/a", []string{"a"}, 3)
	found(t, n, "/a/", []string{"a"}, 3)
	found(t, n, "/a/b", []string{"a", "b"}, 2)
	found(t, n, "/a/b/", []string{"a", "b"}, 2)
	found(t, n, "/a/b/c", []string{"a", "b/c"}, 2)
	found(t, n, "/a/b/c/", []string{"a", "b/c"}, 2)
	found(t, n, "/a/b/c/d", []string{"a", "b/c/d"}, 2)
	found(t, n, "/first/second", []string{"first", "second"}, 2)
	found(t, n, "/first/second/", []string{"first", "second"}, 2)
	found(t, n, "/first/second/third", []string{"third"}, 1)
}

func TestMixedTree(t *testing.T) {
	n := New()

	n.Add("/", 0)
	n.Add("/path/to/nowhere", 1)
	n.Add("/path/:i/nowhere", 2)
	n.Add("/:id/to/nowhere", 3)
	n.Add("/:a/:b", 4)
	n.Add("/not/found", 5)

	found(t, n, "/", nil, 0)
	found(t, n, "/path/to/nowhere", nil, 1)
	found(t, n, "/path/to/nowhere/", nil, 1)
	found(t, n, "/path/from/nowhere", []string{"from"}, 2)
	found(t, n, "/walk/to/nowhere", []string{"walk"}, 3)
	found(t, n, "/path/to/", []string{"path", "to"}, 4)
	found(t, n, "/path/to", []string{"path", "to"}, 4)
	found(t, n, "/not/found", []string{"not", "found"}, 4)
	notfound(t, n, "/path/to/somewhere")
	notfound(t, n, "/path/to/nowhere/else")
	notfound(t, n, "/path")
	notfound(t, n, "/path/")

	notfound(t, n, "")
	notfound(t, n, "xyz")
	notfound(t, n, "/path//to/nowhere")
}

func notfound(t *testing.T, n *Node, p string) {
	if leaf, _ := n.Find(p); leaf != nil {
		t.Errorf("Should not have found: %s", p)
	}
}

func found(t *testing.T, n *Node, p string, expectedExpansions []string, val interface{}) {
	leaf, expansions := n.Find(p)
	if leaf == nil {
		t.Errorf("Didn't find: %s", p)
		return
	}
	if !reflect.DeepEqual(expansions, expectedExpansions) {
		t.Errorf("%s: Wildcard expansions (actual) %v != %v (expected)", p, expansions, expectedExpansions)
	}
	if leaf.Value != val {
		t.Errorf("%s: Value (actual) %v != %v (expected)", p, leaf.Value, val)
	}
}
