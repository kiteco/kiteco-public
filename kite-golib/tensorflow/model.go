package tensorflow

import (
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/core/protobuf"

	proto "github.com/golang/protobuf/proto"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lazy"
	tf "github.com/kiteco/tensorflow/tensorflow/go"
)

// ForceLoadCycle will force a load and unload of the model. One use-case for this is to
// force loading of the model so that it appears in datadeps
var ForceLoadCycle = false

var sessionOptions *tf.SessionOptions
var m sync.Mutex

func init() {
	SetTensorflowThreadpoolSize(runtime.NumCPU())
}

// SetSessionOptions ...
func SetSessionOptions(cfg *protobuf.ConfigProto) error {
	m.Lock()
	defer m.Unlock()
	ser, err := proto.Marshal(cfg)
	if err != nil {
		return errors.Wrapf(err, "error marshaling config")
	}
	sessionOptions = &tf.SessionOptions{Config: ser}
	return nil
}

// GetSessionOptions ...
func GetSessionOptions() *protobuf.ConfigProto {
	m.Lock()
	defer m.Unlock()
	var cfg protobuf.ConfigProto
	if err := proto.Unmarshal(sessionOptions.Config, &cfg); err != nil {
		panic(err)
	}
	return &cfg
}

// GetTensorflowThreadpoolSize returns the number of threads tensorflow is using
func GetTensorflowThreadpoolSize() int {
	return int(GetSessionOptions().InterOpParallelismThreads)
}

// SetTensorflowThreadpoolSize allows to change the number of thread used by tensorflow for model evaluation (default 1)
// Make sure to call this function before loading the model
func SetTensorflowThreadpoolSize(newSize int) {
	cfg := protobuf.ConfigProto{
		// we don't currently ship TF compiled for GPU, but we're thorough regardless
		DeviceCount: map[string]int32{"CPU": int32(newSize), "GPU": 1},

		IntraOpParallelismThreads: int32(newSize),
		InterOpParallelismThreads: int32(newSize),

		// OperationTimeoutInMs is a per-*operation* timeout,
		// which is too granular to be useful, assuming it does what we think it does.
	}
	if err := SetSessionOptions(&cfg); err != nil {
		panic(err)
	}
}

// sized is implemented in kite-golib/fileutil/filemap.go
type sized interface {
	Len() int64
}

// RunCallback is a function that can be called whenever Run is called, with the inputs and results of the model
type RunCallback func(feeds map[string]interface{}, fetches []string, result map[string]interface{}, err error)

// Model wraps a Tensorflow model
type Model struct {
	*lazy.Loader
	session *tf.Session
	graph   *tf.Graph

	// RunCallback, if set, is called whenever Run is called
	RunCallback RunCallback
}

// NewModel loads a Tensorflow model (serialized as a GraphDef proto, frozen to replace variables with constants)
// from the given local/S3 path.
// The Python kite_ml.kite.save.save_frozen_model() function should save models in this format, as well as
// Tensorflow's freeze_graph utility.
func NewModel(path string) (*Model, error) {
	m := &Model{}

	load := func() error {
		r, err := fileutil.NewCachedReader(path)
		if err != nil {
			return err
		}
		defer r.Close()

		graph := tf.NewGraph()
		if s, _ := r.(sized); s != nil {
			err = graph.ImportFromReader(r, int(s.Len()), "")
		} else {
			var data []byte
			data, err = ioutil.ReadAll(r)
			if err != nil {
				return errors.Wrapf(err, "error reading graph definition")
			}
			err = graph.Import(data, "")
		}
		if err != nil {
			return errors.Wrapf(err, "error importing graph")
		}

		// TODO: at this point we have the opportunity to pass in a timeout parameter
		// To do that, we need to create a GraphConfig proto and serialize it
		sess, err := tf.NewSession(graph, sessionOptions)
		if err != nil {
			graph.Delete()
			return errors.Wrapf(err, "error creating session")
		}

		m.graph = graph
		m.session = sess
		return nil
	}

	unload := func() {
		if m.session != nil {
			m.session.Close()
		}
		if m.graph != nil {
			m.graph.Delete()
		}
		m.session = nil
		m.graph = nil
	}

	// Force a load & unload (used by datadeps generation)
	if ForceLoadCycle {
		err := load()
		if err != nil {
			return nil, err
		}
		unload()
	}

	m.Loader = lazy.NewLoader(load, unload)

	return m, nil
}

// Unload the model
func (m *Model) Unload() {
	m.Loader.Unload()
}

// PartialRun wraps a tensorflow partial run
type PartialRun struct {
	pr             *tf.PartialRun
	feeds, fetches map[string]tf.Output
	targets        map[string]*tf.Operation
}

// NewPartialRun ...
func (m *Model) NewPartialRun(feeds, fetches []string, targets []string) (*PartialRun, error) {
	err := m.Loader.LoadAndLock()
	if err != nil {
		return nil, err
	}
	defer m.Loader.Unlock()

	var tfOuts [2]map[string]tf.Output
	var tfOutsSlice [2][]tf.Output
	for i, names := range [2][]string{feeds, fetches} {
		tfOuts[i] = make(map[string]tf.Output)
		for _, name := range names {
			out, err := m.tfOut(name)
			if err != nil {
				return nil, err
			}
			tfOuts[i][name] = out
			tfOutsSlice[i] = append(tfOutsSlice[i], out)
		}
	}

	tfTargets := make(map[string]*tf.Operation)
	var tfTargetsSlice []*tf.Operation
	for _, name := range targets {
		op := m.graph.Operation(name)
		if op == nil {
			return nil, errors.Errorf("unable to find target op '%s'", name)
		}
		tfTargets[name] = op
		tfTargetsSlice = append(tfTargetsSlice, op)
	}

	pr, err := m.session.NewPartialRun(tfOutsSlice[0], tfOutsSlice[1], tfTargetsSlice)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to initialize partial run")
	}

	return &PartialRun{
		pr:      pr,
		feeds:   tfOuts[0],
		fetches: tfOuts[1],
		targets: tfTargets,
	}, nil
}

// Run resumes execution of the graph to compute the requested fetches and
// targets with the provided feeds.
func (p *PartialRun) Run(feeds map[string]interface{}, fetches, targets []string) (map[string]interface{}, error) {
	tfFeeds := make(map[tf.Output]*tf.Tensor)
	for name, val := range feeds {
		out, ok := p.feeds[name]
		if !ok {
			return nil, errors.Errorf("unable to find feed with name '%s'", name)
		}

		tVal, err := tf.NewTensor(val)
		if err != nil {
			return nil, errors.Errorf("error creating tensor for value of '%s': %v", name, err)
		}

		tfFeeds[out] = tVal
	}

	// Cleanup tensors
	defer func() {
		for _, t := range tfFeeds {
			t.Delete()
		}
	}()

	var tfFetches []tf.Output
	for _, name := range fetches {
		out, ok := p.fetches[name]
		if !ok {
			return nil, errors.Errorf("unable to find fetch with name '%s'", name)
		}
		tfFetches = append(tfFetches, out)
	}

	var tfTargets []*tf.Operation
	for _, name := range targets {
		t, ok := p.targets[name]
		if !ok {
			return nil, errors.Errorf("unable to find target with name '%s'", name)
		}
		tfTargets = append(tfTargets, t)
	}

	return runTF(func() ([]*tf.Tensor, error) {
		return p.pr.Run(tfFeeds, tfFetches, tfTargets)
	}, fetches)
}

// OpExists ...
func (m *Model) OpExists(name string) (bool, error) {
	err := m.Loader.LoadAndLock()
	if err != nil {
		return false, err
	}
	defer m.Loader.Unlock()
	for _, op := range m.graph.Operations() {
		if op.Name() == name {
			return true, nil
		}
	}
	return false, nil
}

// Run takes in a map of feed tensors, keyed by the operation names, as well as a slice of operations to fetch.
// As output, it returns a map of output operation names to the resulting output tensors.
func (m *Model) Run(feeds map[string]interface{}, fetches []string) (map[string]interface{}, error) {
	res, err := m.run(feeds, fetches)
	if m.RunCallback != nil {
		m.RunCallback(feeds, fetches, res, err)
	}
	return res, err
}

func (m *Model) run(feeds map[string]interface{}, fetches []string) (map[string]interface{}, error) {
	err := m.Loader.LoadAndLock()
	if err != nil {
		return nil, err
	}
	defer m.Loader.Unlock()

	tfFeeds := make(map[tf.Output]*tf.Tensor)

	for op, val := range feeds {
		out, err := m.tfOut(op)
		if err != nil {
			return nil, err
		}
		tensor, err := tf.NewTensor(val)

		if err != nil {
			return nil, errors.Wrapf(err, "error creating tensor")
		}
		tfFeeds[out] = tensor
	}

	// Cleanup tensors
	defer func() {
		for _, t := range tfFeeds {
			t.Delete()
		}
	}()

	var tfFetches []tf.Output
	for _, op := range fetches {
		out, err := m.tfOut(op)
		if err != nil {
			return nil, err
		}
		tfFetches = append(tfFetches, out)
	}

	return runTF(func() ([]*tf.Tensor, error) {
		return m.session.Run(tfFeeds, tfFetches, nil)
	}, fetches)
}

func (m *Model) tfOut(opName string) (tf.Output, error) {
	op := m.graph.Operation(opName)
	if op == nil {
		return tf.Output{}, errors.Errorf("could not find op with name: %s", opName)
	}

	return tf.Output{
		Op:    op,
		Index: 0,
	}, nil
}

func runTF(run func() ([]*tf.Tensor, error), fetches []string) (map[string]interface{}, error) {
	res, err := run()
	if err != nil {
		return nil, errors.Wrapf(err, "error running model")
	}

	// Cleanup tensors
	defer func() {
		for _, t := range res {
			t.Delete()
		}
	}()

	out := make(map[string]interface{})
	for i, op := range fetches {
		out[op] = res[i].Value()
	}

	return out, nil
}
