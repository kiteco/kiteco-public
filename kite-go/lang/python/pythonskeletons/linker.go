package pythonskeletons

import (
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// --

type linker struct {
	graph        *pythonimports.Graph
	builtins     *pythonimports.BuiltinCache
	cache        map[pythonimports.Hash]*pythonimports.Node
	instances    map[pythonimports.Hash]*pythonimports.Node
	classTmpl    map[string]NodeTemplate
	methodTmpl   map[string]NodeTemplate
	functionTmpl map[string]NodeTemplate
	id           int64
}

func newLinker(graph *pythonimports.Graph, builtins *pythonimports.BuiltinCache) *linker {
	return &linker{
		graph:        graph,
		builtins:     builtins,
		cache:        make(map[pythonimports.Hash]*pythonimports.Node),
		instances:    make(map[pythonimports.Hash]*pythonimports.Node),
		classTmpl:    ClassDefaultAttributes(builtins),
		methodTmpl:   MemberFuncDefaultAttributes(builtins),
		functionTmpl: FuncDefaultAttributes(builtins),
		id:           int64(len(graph.Nodes)),
	}
}

func (l *linker) LinkInstance(ty pythonimports.DottedPath) *pythonimports.Node {
	// check cache
	if node, found := l.instances[ty.Hash]; found {
		return node
	}

	// type node
	tnode := l.Link(ty, pythonimports.Type)
	if tnode == nil {
		return nil
	}

	inst := l.newNode(pythonimports.DottedPath{}, pythonimports.Object, tnode)
	l.instances[ty.Hash] = inst
	return inst
}

func (l *linker) Link(path pythonimports.DottedPath, kind pythonimports.Kind) *pythonimports.Node {
	// check cache
	if node, found := l.cache[path.Hash]; found {
		return node
	}
	return l.linkImpl(path, kind)
}

func (l *linker) linkImpl(path pythonimports.DottedPath, kind pythonimports.Kind) *pythonimports.Node {
	// deal with top level module
	if len(path.Parts) == 1 {
		mod := l.graph.PkgToNode[path.Parts[0]]
		if mod == nil {
			log.Println("creating new top level module:", path.String())
			mod = l.newNode(path, kind, l.builtins.Module)
			l.graph.PkgToNode[path.Parts[0]] = mod
			l.graph.Root.Members[path.Parts[0]] = mod
		}
		return mod
	}

	// find parent
	parent := l.graph.Root
	for i, part := range path.Parts[:len(path.Parts)-1] {
		next := parent.Members[part]
		if next == nil {
			partial := pythonimports.NewPath(path.Parts[:i+1]...)
			// TODO(juan): kind of hacky...
			if index().Types[partial.Hash] != nil {
				next = l.Link(partial, pythonimports.Type)
			} else {
				// NOTE(juan): all ancestors for a path are assumed to be modules
				next = l.Link(partial, pythonimports.Module)
			}
			// attach new node to graph
			parent.Members[part] = next
		}
		parent = next
	}

	child := parent.Members[path.Last()]
	if child != nil {
		// allow clients to overide missing canonical names
		child.CanonicalName = path
		return child
	}

	switch {
	case parent.Classification == pythonimports.Type && kind == pythonimports.Function:
		child = l.newNodeFromTemplate(path, pythonimports.Function, l.builtins.Function, l.methodTmpl)
	case kind == pythonimports.Function:
		child = l.newNodeFromTemplate(path, pythonimports.Function, l.builtins.Function, l.functionTmpl)
	case kind == pythonimports.Type:
		child = l.newNodeFromTemplate(path, pythonimports.Type, l.builtins.Type, l.classTmpl)
	case kind == pythonimports.Module:
		child = l.newNode(path, pythonimports.Module, l.builtins.Module)
	default:
		panic(fmt.Sprintf("error building node for %s, this should not happen", path.String()))
	}
	log.Println("created new node:", child.CanonicalName.String())
	parent.Members[path.Last()] = child
	return child
}

func (l *linker) newNode(name pythonimports.DottedPath, kind pythonimports.Kind, typ *pythonimports.Node) *pythonimports.Node {
	l.id++
	node := &pythonimports.Node{
		NodeInfo: pythonimports.NodeInfo{
			CanonicalName:  name,
			Classification: kind,
			Origin:         pythonimports.GlobalGraph,
			ID:             l.id,
		},
		Type:    typ,
		Members: make(map[string]*pythonimports.Node),
	}

	if !node.CanonicalName.Empty() {
		l.cache[node.CanonicalName.Hash] = node
	}
	return node
}

func (l *linker) newNodeFromTemplate(name pythonimports.DottedPath, kind pythonimports.Kind, typ *pythonimports.Node,
	defaults map[string]NodeTemplate) *pythonimports.Node {

	node := l.newNode(name, kind, typ)
	for name, attr := range defaults {
		node.Members[name] = l.newNode(pythonimports.DottedPath{}, attr.Kind, attr.Type)
	}
	return node
}
