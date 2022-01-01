package pythongraph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
)

// nameSet represents a set of names along with the order in which
// they were evaluated by the propagator, in particular
// nameSet.Set[name] == the order in which name was evaluated by the propagator.
type nameSet struct {
	set    map[*pythonast.NameExpr]int
	sorted []*pythonast.NameExpr
}

func newNameSet() *nameSet {
	return &nameSet{}
}

func (n *nameSet) Copy() *nameSet {
	if n.Empty() {
		return nil
	}

	setCopy := make(map[*pythonast.NameExpr]int, len(n.set))
	for k, v := range n.set {
		setCopy[k] = v
	}
	return &nameSet{set: setCopy}
}

func (n *nameSet) Names() []*pythonast.NameExpr {
	if n.Empty() {
		return nil
	}

	if n.sorted != nil {
		return n.sorted
	}

	names := make([]*pythonast.NameExpr, 0, n.Len())
	for name := range n.Set() {
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		orderi, _ := n.Get(names[i])
		orderj, _ := n.Get(names[j])
		return orderi < orderj
	})

	n.sorted = names

	return names
}

func (n *nameSet) Equals(other *nameSet) bool {
	if other == nil && n == nil {
		return true
	}

	if other == nil {
		return false
	}

	if n == nil {
		return false
	}

	if n.Len() != other.Len() {
		return false
	}

	for name := range n.Set() {
		if _, ok := other.Get(name); !ok {
			return false
		}
	}
	return true
}

func (n *nameSet) String() string {
	names := n.Names()

	var parts []string
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("[%d:%d] %s", name.Begin(), name.End(), name.Ident.String()))
	}
	return strings.Join(parts, " , ")
}

func (n *nameSet) Set() map[*pythonast.NameExpr]int {
	if n == nil {
		return nil
	}
	return n.set
}

func (n *nameSet) Empty() bool {
	return n.Len() == 0
}

func (n *nameSet) Add(name *pythonast.NameExpr, order int) bool {
	if _, ok := n.Get(name); ok {
		return false
	}

	n.sorted = nil
	if n.set == nil {
		n.set = make(map[*pythonast.NameExpr]int)
	}
	n.set[name] = order
	return true
}

func (n *nameSet) Get(name *pythonast.NameExpr) (int, bool) {
	if n == nil {
		return 0, false
	}

	order, ok := n.set[name]
	return order, ok
}

func (n *nameSet) Contains(name *pythonast.NameExpr) bool {
	_, ok := n.Get(name)
	return ok
}

func (n *nameSet) Delete(name *pythonast.NameExpr) {
	if n == nil {
		return
	}

	if _, ok := n.set[name]; ok {
		delete(n.set, name)
		n.sorted = nil
	}
}

func (n *nameSet) Len() int {
	if n == nil {
		return 0
	}
	return len(n.Set())
}
