package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"

	arg "github.com/alexflint/go-arg"
	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// These are the class attributes that exist on almost any object, so for size reasons
// omit them from the import graph
var commonMembers = map[string]struct{}{
	"__class__":        struct{}{},
	"__delattr__":      struct{}{},
	"__dict__":         struct{}{},
	"__doc__":          struct{}{},
	"__format__":       struct{}{},
	"__getattribute__": struct{}{},
	"__hash__":         struct{}{},
	"__module__":       struct{}{},
	"__new__":          struct{}{},
	"__reduce__":       struct{}{},
	"__reduce_ex__":    struct{}{},
	"__repr__":         struct{}{},
	"__setattr__":      struct{}{},
	"__sizeof__":       struct{}{},
	"__str__":          struct{}{},
	"__subclasshook__": struct{}{},
	"__weakref__":      struct{}{},
}

type memberPair struct {
	attr     string
	masterID int64
	otherID  int64
}

// Compute the intersection of the keys of two maps
func intersectKeys(masterMembers, otherMembers []pythonimports.FlatMember) []memberPair {
	masterMap := make(map[string]int64)
	for _, m := range masterMembers {
		masterMap[m.Attr] = m.NodeID
	}

	var pairs []memberPair
	for _, m := range otherMembers {
		if masterID, found := masterMap[m.Attr]; found {
			pairs = append(pairs, memberPair{
				attr:     m.Attr,
				masterID: masterID,
				otherID:  m.NodeID,
			})
		}
	}
	return pairs
}

// ---

type pathConflict struct {
	FirstPath      string
	SecondPath     string
	FirstMasterID  int64
	SecondMasterID int64
	OtherID        int64
}

type valueConflict struct {
	Path     string
	MasterID int64
	OtherID  int64
}

type aligner struct {
	masterIDByOtherID map[int64]int64
	pathByOtherID     map[int64]string
	master            map[int64]*node
	other             map[int64]*node
	conflicts         []interface{}
}

func newAligner(master, other map[int64]*node) *aligner {
	return &aligner{
		master:            master,
		other:             other,
		masterIDByOtherID: make(map[int64]int64),
		pathByOtherID:     make(map[int64]string),
	}
}

func (a *aligner) align(masterID, otherID int64, path string) bool {
	// do we already have this correspondence?
	if origMasterID, seen := a.masterIDByOtherID[otherID]; seen {
		if origMasterID != masterID {
			a.conflicts = append(a.conflicts, &pathConflict{
				FirstPath:      a.pathByOtherID[otherID],
				SecondPath:     path,
				FirstMasterID:  origMasterID,
				SecondMasterID: masterID,
				OtherID:        otherID,
			})
			return false
		}
		return true
	}

	// attempt to recurse - it's normal for nodes to be missing because import exploration deliberately
	// skips certain nodes
	masterNode, masterPresent := a.master[masterID]
	otherNode, otherPresent := a.other[otherID]
	if !masterPresent || !otherPresent {
		// One or the other node is not present - we cannot deduce anything more
		// about whether these nodes are the same object
		a.masterIDByOtherID[otherID] = masterID
		a.pathByOtherID[otherID] = path
		return true
	}

	// if both have canonical names but they are not the same then reject the match
	if !masterNode.CanonicalName.Empty() &&
		!otherNode.CanonicalName.Empty() &&
		masterNode.CanonicalName.Hash != otherNode.CanonicalName.Hash {

		a.conflicts = append(a.conflicts, &valueConflict{
			Path:     path,
			MasterID: masterID,
			OtherID:  otherID,
		})
		return false
	}

	// compare str, repr, and classification for the two objects
	if masterNode.Classification != otherNode.Classification ||
		masterNode.StrHash != otherNode.StrHash ||
		masterNode.ReprHash != otherNode.ReprHash {

		a.conflicts = append(a.conflicts, &valueConflict{
			Path:     path,
			MasterID: masterID,
			OtherID:  otherID,
		})
		return false
	}

	// accept the match
	a.masterIDByOtherID[otherID] = masterID
	a.pathByOtherID[otherID] = path

	// do not recurse into the attributes of objects, descriptors, or functions
	if masterNode.Classification != pythonimports.Module && masterNode.Classification != pythonimports.Type {
		return true
	}

	// align each child
	for _, pair := range intersectKeys(masterNode.Members, otherNode.Members) {
		// ignore attributes that start with "_" for now
		if !strings.HasPrefix(pair.attr, "_") {
			a.align(pair.masterID, pair.otherID, path+"."+pair.attr)
		}
	}
	return true
}

func (a *aligner) translateOrAssignID(otherID int64, generateID func() int64) (int64, bool) {
	masterID, wasPresent := a.masterIDByOtherID[otherID]
	if !wasPresent {
		// this node does not yet have a master ID so assign a new ID
		masterID = generateID()
		a.masterIDByOtherID[otherID] = masterID
	}
	return masterID, wasPresent
}

// ---

// node represents an import graph node with some extra data used only during merging
type node struct {
	pythonimports.FlatNode
	StrHash  uint64 // StrHash is a hash of the str() of the object
	ReprHash uint64 // ReprHash is a hash of the repr() of the object
}

// flatEdge represents an attribute on a node, using IDs to reference other nodes
// so that the structure is acyclic for serialization.
type flatEdge struct {
	NodeID int64 `json:"node_id"`
}

// record is the struct that we read from the input shards
type record struct {
	pythonimports.FlatNode
	pythonimports.NodeStrings
	ClassificationStr string                 `json:"classification"` // supercedes FlatNode.Classification
	MembersMap        map[string]flatEdge    `json:"members"`        // supercedes FlatNode.Members
	CanonicalNameStr  string                 `json:"canonical_name"`
	ArgSpec           *pythonimports.ArgSpec `json:"argspec"`
}

// graph is a collection of nodes representing the Python import graph.
type graph struct {
	Nodes map[int64]*node
	IDs   map[int64]struct{}
}

// newGraph creates a graph
func newGraph() *graph {
	return &graph{
		Nodes: make(map[int64]*node),
		IDs:   make(map[int64]struct{}),
	}
}

// convert a classification string to a Kind
func nodeClassFromString(s string) (pythonimports.Kind, error) {
	switch s {
	case "function":
		return pythonimports.Function, nil
	case "type":
		return pythonimports.Type, nil
	case "module":
		return pythonimports.Module, nil
	case "descriptor":
		return pythonimports.Descriptor, nil
	case "object":
		return pythonimports.Object, nil
	default:
		return pythonimports.Kind(0), fmt.Errorf("unrecognized classifcation: '%s'", s)
	}
}

// load a graph from a json file
func loadNodes(path string) (map[int64]*node, []*record, error) {
	var records []*record
	nodes := make(map[int64]*node)
	err := serialization.Decode(path, func(record *record) {
		// process canonical name
		record.CanonicalName = pythonimports.NewDottedPath(record.CanonicalNameStr)

		// process classification
		classification, err := nodeClassFromString(record.ClassificationStr)
		if err != nil {
			log.Println(err)
			return
		}
		record.Classification = classification

		// process members
		if record.MembersMap != nil {
			for attr, edge := range record.MembersMap {
				// remove very common members to save space
				if _, common := commonMembers[attr]; !common {
					record.Members = append(record.Members, pythonimports.FlatMember{
						Attr:   attr,
						NodeID: edge.NodeID,
					})
				}
			}
		}

		// add to slices
		records = append(records, record)
		nodes[record.ID] = &node{
			FlatNode: record.FlatNode,
			StrHash:  spooky.Hash64([]byte(record.Str)),
			ReprHash: spooky.Hash64([]byte(record.Repr)),
		}
	})
	return nodes, records, err
}

// newID selects an unused ID, adds it to the list of used IDs, and then returns it
func (g *graph) newID() int64 {
	for i := 0; i < 1000; i++ {
		ID := rand.Int63()
		if _, clash := g.IDs[ID]; !clash {
			g.IDs[ID] = struct{}{}
			return ID
		}
	}
	panic("failed to generate a non-clashing ID after 1000 attempts")
}

// update integrates another graph into this graph
func (g *graph) update(other map[int64]*node) (forwardID map[int64]int64, err error) {
	masterIDByName := make(map[string]int64)
	for id, node := range g.Nodes {
		if !node.CanonicalName.Empty() {
			masterIDByName[node.CanonicalName.String()] = id
		}
	}

	// compute the ID map
	a := newAligner(g.Nodes, other)
	var numCorrespondingName, numConflictingNames int
	for otherID, otherNode := range other {
		if otherNode.CanonicalName.Empty() {
			continue
		}
		if masterID, present := masterIDByName[otherNode.CanonicalName.String()]; present {
			numCorrespondingName++
			rootPath := "[[" + otherNode.CanonicalName.String() + "]]"
			if !a.align(masterID, otherID, rootPath) {
				numConflictingNames++

				// for now just ignore the conflict
				a.masterIDByOtherID[otherID] = masterID
			}
		}
	}

	log.Printf("Found %d name-based correspondence, extrapolated to %d incomming IDs",
		numCorrespondingName, len(a.masterIDByOtherID))
	log.Printf("%d of %d name-based correspondences failed to align",
		numConflictingNames, numCorrespondingName)
	log.Printf("Encountered %d other conflicts", len(a.conflicts))

	// transform the nodes from other into master
	var numNewAttrs, numNewNodes, numOrphans int
	for otherID, otherNode := range other {
		masterID, wasPresent := a.translateOrAssignID(otherID, g.newID)
		if !wasPresent {
			numOrphans++
		}

		// check whether the node is present
		masterNode, present := g.Nodes[masterID]
		if present {
			masterMembers := make(map[string]struct{})
			for _, member := range masterNode.Members {
				masterMembers[member.Attr] = struct{}{}
			}
			// node already present in master: insert any children from otherNode that
			// are not already in masterNode
			for _, otherMember := range otherNode.Members {
				if _, inMaster := masterMembers[otherMember.Attr]; !inMaster {
					numNewAttrs++
					ID, _ := a.translateOrAssignID(otherMember.NodeID, g.newID)
					masterNode.Members = append(masterNode.Members, pythonimports.FlatMember{
						Attr:   otherMember.Attr,
						NodeID: ID,
					})
				}
			}
		} else {
			// need to add this node: convert all IDs from the other domain to the master domain
			numNewNodes++
			otherNode.ID = masterID
			otherNode.TypeID, _ = a.translateOrAssignID(otherNode.TypeID, g.newID)
			for i := range otherNode.Members {
				m := &otherNode.Members[i]
				m.NodeID, _ = a.translateOrAssignID(m.NodeID, g.newID)
			}
			g.Nodes[masterID] = otherNode
		}
	}

	log.Printf("Added %d new nodes", numNewNodes)
	log.Printf("Added %d potential orphans", numOrphans)
	log.Printf("Added %d new members to existing nodes", numNewAttrs)
	log.Printf("Master graph now has %d nodes", len(g.Nodes))

	return a.masterIDByOtherID, nil
}

func verifyCanonicalNames(nodes map[int64]*node) int {
	var flats []*pythonimports.FlatNode
	for _, n := range nodes {
		flats = append(flats, &n.FlatNode)
	}
	graph := pythonimports.NewGraphFromNodes(flats)

	var dups int
	for _, n := range nodes {
		if n.CanonicalName.Empty() {
			continue
		}

		// first check if canonical name is in the graph
		resolvedNode, err := graph.Navigate(n.CanonicalName)
		if err != nil {
			continue
		}

		// check if the resolved node is the same one as we started out with
		if original, ok := nodes[resolvedNode.ID]; ok {
			if original.CanonicalName.Equals(n.CanonicalName.String()) && original != n {
				n.CanonicalName = pythonimports.DottedPath{}
				dups++
			}
		}
	}
	return dups
}

func main() {
	var args struct {
		Shards   []string
		Output   string
		Strings  string
		ArgSpecs string
	}
	arg.MustParse(&args)

	if len(args.Shards) == 0 {
		log.Fatalln("Usage: merge-import-graphs INPUT1.JSON INPUT2.JSON ...")
	}

	// Open encoders to fail fast
	var err error
	var graphEnc, stringsEnc, argSpecsEnc *serialization.EncodeCloser
	if args.Output != "" {
		graphEnc, err = serialization.NewEncoder(args.Output)
		if err != nil {
			log.Fatalf("Error opening graph output file: %v\n", err)
		}
		defer graphEnc.Close()
	}
	if args.Strings != "" {
		stringsEnc, err = serialization.NewEncoder(args.Strings)
		if err != nil {
			log.Fatalf("Error opening strings output file: %v\n", err)
		}
		defer stringsEnc.Close()
	}
	if args.ArgSpecs != "" {
		argSpecsEnc, err = serialization.NewEncoder(args.ArgSpecs)
		if err != nil {
			log.Fatalf("Error opening arg specs output file: %v\n", err)
		}
		defer argSpecsEnc.Close()
	}

	// Initialize the master graph
	master := newGraph()

	// Merge each subsequent graph
	var numArgSpecs, numUnreadableFiles, numDups int
	seenIDs := make(map[int64]bool)
	for i, path := range args.Shards {
		log.Println() // helps with readability of logs
		log.Printf("Loading %s (%d of %d)", path, i+1, len(args.Shards))

		shard, records, err := loadNodes(path)
		if err != nil {
			log.Printf("Ignoring %s: %v", path, err)
			numUnreadableFiles++
			continue
		}
		log.Printf("Merging %d nodes from %s:", len(shard), path)

		numDups += verifyCanonicalNames(shard)

		forwardIDs, err := master.update(shard)
		if err != nil {
			log.Println(err)
		}
		log.Println("Done with", path)

		// write out the metadata
		for _, record := range records {
			masterID, found := forwardIDs[record.ID]
			if !found || seenIDs[masterID] {
				continue
			}
			seenIDs[masterID] = true

			if stringsEnc != nil {
				strings := record.NodeStrings
				strings.NodeID = masterID
				stringsEnc.Encode(strings)
			}

			if argSpecsEnc != nil && record.ArgSpec != nil {
				numArgSpecs++
				argspec := record.ArgSpec
				argspec.NodeID = masterID
				argSpecsEnc.Encode(argspec)
			}
		}
	}

	if numUnreadableFiles > 0 {
		log.Printf("%d (of %d) files were unreadable", numUnreadableFiles, len(args.Shards))
	}
	if numDups > 0 {
		log.Printf("Cleared %d canonical names due to dup nodes\n", numDups)
		log.Printf("Total nodes: %d\n", len(master.Nodes))
	}

	// Some packages add nodes to builtins. We can fix this by
	// removing anything within builtins that starts with "_"
	log.Println("Removing bogus nodes from builtin package...")
	for _, node := range master.Nodes {
		if node.CanonicalName.String() == "builtins" {
			var filtered []pythonimports.FlatMember
			for _, member := range node.Members {
				if !strings.HasPrefix(member.Attr, "_") {
					filtered = append(filtered, member)
				}
			}
			node.Members = filtered
			break
		}
	}

	// Link instance attributes back to type attributes
	for _, node := range master.Nodes {
		if node.Classification != pythonimports.Object {
			continue
		}

		typeNode := master.Nodes[node.TypeID]
		if typeNode == nil {
			continue
		}

		typeMembers := make(map[string]int64)
		for _, member := range typeNode.Members {
			typeMembers[member.Attr] = member.NodeID
		}

		for i, member := range node.Members {
			if master.Nodes[member.NodeID] != nil {
				continue
			}

			if typeMemberID, found := typeMembers[member.Attr]; found {
				if master.Nodes[typeMemberID] != nil {
					node.Members[i].NodeID = typeMemberID
				}
			}
		}
	}

	// Compute some simple statistics on the final graph
	log.Println("Enumerating graph...")
	var numNamed int
	var queue []*node
	reachable := make(map[int64]struct{})
	for _, node := range master.Nodes {
		if !node.CanonicalName.Empty() {
			reachable[node.ID] = struct{}{}
			queue = append(queue, node)
			numNamed++
		}
	}

	// Compute the set of reachable nodes
	log.Println("Running breadth-first search...")
	var pivot *node
	var neighbors []int64
	for len(queue) > 0 {
		pivot, queue = queue[0], queue[1:]
		neighbors = neighbors[:0]
		neighbors = append(neighbors, pivot.TypeID)
		for _, edge := range pivot.Members {
			neighbors = append(neighbors, edge.NodeID)
		}
		for _, neighbor := range neighbors {
			if _, seen := reachable[neighbor]; seen {
				continue
			}
			if node, present := master.Nodes[neighbor]; present {
				queue = append(queue, node)
				reachable[neighbor] = struct{}{}
			}
		}
	}
	log.Printf("There are %d reachable IDs (of %d)", len(reachable), len(master.Nodes))

	var nodes []*node
	for id := range reachable {
		nodes = append(nodes, master.Nodes[id])
	}

	log.Println("Wrote auxialiary strings to", args.Strings)
	log.Println("Wrote arg specs to", args.ArgSpecs)

	if graphEnc != nil {
		for _, node := range nodes {
			err = graphEnc.Encode(node.FlatNode)
			if err != nil {
				log.Println(err)
			}
		}
		log.Println("Wrote graph to", args.Output)
	}

	log.Printf("  %d nodes with a canonical name", numNamed)
	log.Printf("  %d nodes reachable from a named node", len(reachable))
	log.Printf("  %d nodes with arg specs", numArgSpecs)
	log.Printf("  %d nodes total", len(master.Nodes))
}
