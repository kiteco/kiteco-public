package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	queueQueueQSizeAddr             = SplitAddress("Queue.Queue.qsize")
	queueQueueEmptyAddr             = SplitAddress("Queue.Queue.empty")
	queueQueueFullAddr              = SplitAddress("Queue.Queue.full")
	queueQueuePutAddr               = SplitAddress("Queue.Queue.put")
	queueQueuePutNoWaitAddr         = SplitAddress("Queue.Queue.put_nowait")
	queueQueueGetAddr               = SplitAddress("Queue.Queue.get")
	queueQueueGetNoWaitAddr         = SplitAddress("Queue.Queue.get_nowait")
	queueQueueTaskDoneAddr          = SplitAddress("Queue.Queue.task_done")
	queueQueueJoinAddr              = SplitAddress("Queue.Queue.join")
	queueLifoQueueQSizeAddr         = SplitAddress("Queue.LifoQueue.qsize")
	queueLifoQueueEmptyAddr         = SplitAddress("Queue.LifoQueue.empty")
	queueLifoQueueFullAddr          = SplitAddress("Queue.LifoQueue.full")
	queueLifoQueuePutAddr           = SplitAddress("Queue.LifoQueue.put")
	queueLifoQueuePutNoWaitAddr     = SplitAddress("Queue.LifoQueue.put_nowait")
	queueLifoQueueGetAddr           = SplitAddress("Queue.LifoQueue.get")
	queueLifoQueueGetNoWaitAddr     = SplitAddress("Queue.LifoQueue.get_nowait")
	queueLifoQueueTaskDoneAddr      = SplitAddress("Queue.LifoQueue.task_done")
	queueLifoQueueJoinAddr          = SplitAddress("Queue.LifoQueue.join")
	queuePriorityQueueQSizeAddr     = SplitAddress("Queue.PriorityQueue.qsize")
	queuePriorityQueueEmptyAddr     = SplitAddress("Queue.PriorityQueue.empty")
	queuePriorityQueueFullAddr      = SplitAddress("Queue.PriorityQueue.full")
	queuePriorityQueuePutAddr       = SplitAddress("Queue.PriorityQueue.put")
	queuePriorityQueuePutNoWaitAddr = SplitAddress("Queue.PriorityQueue.put_nowait")
	queuePriorityQueueGetAddr       = SplitAddress("Queue.PriorityQueue.get")
	queuePriorityQueueGetNoWaitAddr = SplitAddress("Queue.PriorityQueue.get_nowait")
	queuePriorityQueueTaskDoneAddr  = SplitAddress("Queue.PriorityQueue.task_done")
	queuePriorityQueueJoinAddr      = SplitAddress("Queue.PriorityQueue.join")
)

// Queue contains values representing the members of the Queue package
var Queue struct {
	Queue         Value
	LifoQueue     Value
	PriorityQueue Value
}

func init() {
	Queue.Queue = newRegType("Queue.Queue", func(args Args) Value { return NewQueue(nil) }, Builtins.Object, map[string]Value{
		"qsize":      nil,
		"empty":      nil,
		"full":       nil,
		"put":        nil,
		"put_nowait": nil,
		"get":        nil,
		"get_nowait": nil,
		"task_done":  nil,
		"join":       nil,
	})

	Queue.LifoQueue = newRegType("Queue.LifoQueue", func(args Args) Value { return NewLifoQueue(nil) }, Queue.Queue, map[string]Value{
		"qsize":      nil,
		"empty":      nil,
		"full":       nil,
		"put":        nil,
		"put_nowait": nil,
		"get":        nil,
		"get_nowait": nil,
		"task_done":  nil,
		"join":       nil,
	})
	Queue.PriorityQueue = newRegType("Queue.PriorityQueue", func(args Args) Value { return NewPriorityQueue(nil) }, Queue.Queue, map[string]Value{
		"qsize":      nil,
		"empty":      nil,
		"full":       nil,
		"put":        nil,
		"put_nowait": nil,
		"get":        nil,
		"get_nowait": nil,
		"task_done":  nil,
		"join":       nil,
	})
}

// QueueInstance represents an instance of a Queue.Queue
type QueueInstance struct {
	Element Value
}

// NewQueue creates a Queue.Queue instance
func NewQueue(elem Value) Value {
	return QueueInstance{elem}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v QueueInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v QueueInstance) Type() Value { return Queue.Queue }

// Address gets the fully qualified path to this value in the import graph
func (v QueueInstance) Address() Address { return Address{} }

// attr looks up an attribute on this value
func (v QueueInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "qsize":
		return SingleResult(BoundMethod{queueQueueQSizeAddr, func(args Args) Value { return IntInstance{} }}, v), nil
	case "empty":
		return SingleResult(BoundMethod{queueQueueEmptyAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "full":
		return SingleResult(BoundMethod{queueQueueFullAddr, func(args Args) Value { return BoolInstance{} }}, v), nil
	case "put":
		return SingleResult(BoundMethod{queueQueuePutAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "put_nowait":
		return SingleResult(BoundMethod{queueQueuePutNoWaitAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "get":
		return SingleResult(BoundMethod{queueQueueGetAddr, func(args Args) Value { return v.Element }}, v), nil
	case "get_nowait":
		return SingleResult(BoundMethod{queueQueueGetNoWaitAddr, func(args Args) Value { return v.Element }}, v), nil
	case "task_done":
		return SingleResult(BoundMethod{queueQueueTaskDoneAddr, func(args Args) Value { return BoolInstance{} }}, v), nil
	case "join":
		return SingleResult(BoundMethod{queueQueueJoinAddr, func(args Args) Value { return Builtins.None }}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v QueueInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(QueueInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat version of this value
func (v QueueInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Queue = &FlatQueue{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v QueueInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltQueue, v.Element)
}

// String provides a string representation of this value
func (v QueueInstance) String() string {
	return fmt.Sprintf("Queue.Queue{%v}", v.Element)
}

// FlatQueue is the representation of Queue used for serialization
type FlatQueue struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatQueue) Inflate(r *Inflater) Value {
	return NewQueue(r.Inflate(f.Element))
}

// LifoQueueInstance represents an instance of a Queue.LifoQueue
type LifoQueueInstance struct {
	Element Value
}

// NewLifoQueue creates a Queue.LifoQueue instance
func NewLifoQueue(elem Value) Value {
	return LifoQueueInstance{elem}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v LifoQueueInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v LifoQueueInstance) Type() Value { return Queue.LifoQueue }

// Address gets the fully qualified path to this value in the import graph
func (v LifoQueueInstance) Address() Address { return Address{} }

// attr looks up an attribute on this value
func (v LifoQueueInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "qsize":
		return SingleResult(BoundMethod{queueLifoQueueQSizeAddr, func(args Args) Value { return IntInstance{} }}, v), nil
	case "empty":
		return SingleResult(BoundMethod{queueLifoQueueEmptyAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "full":
		return SingleResult(BoundMethod{queueLifoQueueFullAddr, func(args Args) Value { return BoolInstance{} }}, v), nil
	case "put":
		return SingleResult(BoundMethod{queueLifoQueuePutAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "put_nowait":
		return SingleResult(BoundMethod{queueLifoQueuePutNoWaitAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "get":
		return SingleResult(BoundMethod{queueLifoQueueGetAddr, func(args Args) Value { return v.Element }}, v), nil
	case "get_nowait":
		return SingleResult(BoundMethod{queueLifoQueueGetNoWaitAddr, func(args Args) Value { return v.Element }}, v), nil
	case "task_done":
		return SingleResult(BoundMethod{queueLifoQueueTaskDoneAddr, func(args Args) Value { return BoolInstance{} }}, v), nil
	case "join":
		return SingleResult(BoundMethod{queueLifoQueueJoinAddr, func(args Args) Value { return Builtins.None }}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v LifoQueueInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(LifoQueueInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat version of this value
func (v LifoQueueInstance) Flatten(f *FlatValue, r *Flattener) {
	f.LifoQueue = &FlatLifoQueue{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v LifoQueueInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltLifoQueue, v.Element)
}

// String provides a string representation of this value
func (v LifoQueueInstance) String() string {
	return fmt.Sprintf("Queue.LifoQueue{%v}", v.Element)
}

// FlatLifoQueue is the representation of Queue used for serialization
type FlatLifoQueue struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatLifoQueue) Inflate(r *Inflater) Value {
	return NewLifoQueue(r.Inflate(f.Element))
}

// PriorityQueueInstance represents an instance of a Queue.PriorityQueue
type PriorityQueueInstance struct {
	Element Value
}

// NewPriorityQueue creates a Queue.Queue instance
func NewPriorityQueue(elem Value) Value {
	return PriorityQueueInstance{elem}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v PriorityQueueInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v PriorityQueueInstance) Type() Value { return Queue.PriorityQueue }

// Address gets the fully qualified path to this value in the import graph
func (v PriorityQueueInstance) Address() Address { return Address{} }

// attr looks up an attribute on this value
func (v PriorityQueueInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "qsize":
		return SingleResult(BoundMethod{queuePriorityQueueQSizeAddr, func(args Args) Value { return IntInstance{} }}, v), nil
	case "empty":
		return SingleResult(BoundMethod{queuePriorityQueueEmptyAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "full":
		return SingleResult(BoundMethod{queuePriorityQueueFullAddr, func(args Args) Value { return BoolInstance{} }}, v), nil
	case "put":
		return SingleResult(BoundMethod{queuePriorityQueuePutAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "put_nowait":
		return SingleResult(BoundMethod{queuePriorityQueuePutNoWaitAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "get":
		return SingleResult(BoundMethod{queuePriorityQueueGetAddr, func(args Args) Value { return v.Element }}, v), nil
	case "get_nowait":
		return SingleResult(BoundMethod{queuePriorityQueueGetNoWaitAddr, func(args Args) Value { return v.Element }}, v), nil
	case "task_done":
		return SingleResult(BoundMethod{queuePriorityQueueTaskDoneAddr, func(args Args) Value { return BoolInstance{} }}, v), nil
	case "join":
		return SingleResult(BoundMethod{queuePriorityQueueJoinAddr, func(args Args) Value { return Builtins.None }}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v PriorityQueueInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(PriorityQueueInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat version of this value
func (v PriorityQueueInstance) Flatten(f *FlatValue, r *Flattener) {
	f.PriorityQueue = &FlatPriorityQueue{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v PriorityQueueInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltPriorityQueue, v.Element)
}

// String provides a string representation of this value
func (v PriorityQueueInstance) String() string {
	return fmt.Sprintf("PriorityQueue.Queue{%v}", v.Element)
}

// FlatPriorityQueue is the representation of Queue used for serialization
type FlatPriorityQueue struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatPriorityQueue) Inflate(r *Inflater) Value {
	return NewPriorityQueue(r.Inflate(f.Element))
}
