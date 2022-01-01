package tfserving

import (
	"crypto/tls"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/kiteserver"
	tf_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/core/framework"
	serving_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/serving"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client is a tfserving compatible lexical client
type Client struct {
	addr   string
	model  string
	conn   *grpc.ClientConn
	client serving_proto.PredictionServiceClient
}

// NewClient returns a client connected to the provided grpc server address
func NewClient(serverURL, model string) (*Client, error) {
	parsedURL, err := kiteserver.ParseKiteServerURL(serverURL)
	if err != nil {
		return nil, err
	}

	// use TLS for https:// urls, disable TLS for all other
	var opts []grpc.DialOption
	if parsedURL.Scheme == "https" {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(parsedURL.Host, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot connect to tfserving grpc server")
	}

	client := serving_proto.NewPredictionServiceClient(conn)

	return &Client{
		addr:   parsedURL.Host,
		model:  model,
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the underlying connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// SearchPrefixSuffix runs the prefix suffix lexical model
func (c *Client) SearchPrefixSuffix(kctx kitectx.Context, before, after, prefixIDs []int64) ([][]int64, [][]float32, error) {
	inputs := c.searchPlaceholders(prefixIDs)
	inputs["context_before"] = c.contextPlaceholder(before)
	inputs["context_after"] = c.contextPlaceholder(after)

	return c.search(kctx, inputs)
}

// Search runs the lexical model on the provided context
func (c *Client) Search(kctx kitectx.Context, context, mask, prefixIDs []int64) ([][]int64, [][]float32, error) {
	inputs := c.searchPlaceholders(prefixIDs)
	inputs["context"] = c.contextPlaceholder(context)
	inputs["context_mask"] = c.contextPlaceholder(mask)

	return c.search(kctx, inputs)
}

func (c *Client) search(kctx kitectx.Context, inputs map[string]*tf_proto.TensorProto) ([][]int64, [][]float32, error) {
	request := &serving_proto.PredictRequest{
		ModelSpec: &serving_proto.ModelSpec{
			Name:          c.model,
			SignatureName: "serving_default",
		},
		Inputs:       inputs,
		OutputFilter: []string{"search_results", "search_probs"},
	}

	resp, err := doRequest(kctx, c.client, request)
	if err != nil {
		return nil, nil, err
	}

	outputs := resp.GetOutputs()
	if outputs == nil {
		return nil, nil, errors.Errorf("no outputs")
	}

	err = ensureOutputTensor(outputs, "search_results", 3)
	if err != nil {
		return nil, nil, err
	}

	err = ensureOutputTensor(outputs, "search_probs", 3)
	if err != nil {
		return nil, nil, err
	}

	results, err := reshapeInt64(
		outputs["search_results"].Int64Val,
		outputs["search_results"].TensorShape.Dim[1].Size,
		outputs["search_results"].TensorShape.Dim[2].Size,
	)

	probs, err := reshapeFloat32(
		outputs["search_probs"].FloatVal,
		outputs["search_probs"].TensorShape.Dim[1].Size,
		outputs["search_probs"].TensorShape.Dim[2].Size,
	)

	return results, probs, nil
}

// --

func (c *Client) contextPlaceholder(context []int64) *tf_proto.TensorProto {
	return &tf_proto.TensorProto{
		Dtype: tf_proto.DataType_DT_INT64,
		TensorShape: &tf_proto.TensorShapeProto{
			Dim: []*tf_proto.TensorShapeProto_Dim{
				&tf_proto.TensorShapeProto_Dim{
					Size: int64(1),
				},
				&tf_proto.TensorShapeProto_Dim{
					Size: int64(len(context)),
				},
			},
		},
		Int64Val: context,
	}
}

func (c *Client) searchPlaceholders(prefixIDs []int64) map[string]*tf_proto.TensorProto {
	return map[string]*tf_proto.TensorProto{
		"valid_prefix_ids": &tf_proto.TensorProto{
			Dtype: tf_proto.DataType_DT_INT64,
			TensorShape: &tf_proto.TensorShapeProto{
				Dim: []*tf_proto.TensorShapeProto_Dim{
					&tf_proto.TensorShapeProto_Dim{
						Size: int64(1),
					},
					&tf_proto.TensorShapeProto_Dim{
						Size: int64(len(prefixIDs)),
					},
				},
			},
			Int64Val: prefixIDs,
		},
	}
}

func reshapeInt64(input []int64, dim1, dim2 int64) ([][]int64, error) {
	if len(input) != int(dim1*dim2) {
		return nil, errors.Errorf("reshapeInt64: dims mismatch, len:%d, dim1:%d, dim2:%d",
			len(input), dim1, dim2)
	}

	var ret [][]int64
	for i := int64(0); i < dim1; i++ {
		ret = append(ret, input[dim2*i:dim2*i+dim2])
	}

	return ret, nil
}

func reshapeFloat32(input []float32, dim1, dim2 int64) ([][]float32, error) {
	if len(input) != int(dim1*dim2) {
		return nil, errors.Errorf("reshapeInt64: dims mismatch, len:%d, dim1:%d, dim2:%d",
			len(input), dim1, dim2)
	}

	var ret [][]float32
	for i := int64(0); i < dim1; i++ {
		ret = append(ret, input[dim2*i:dim2*i+dim2])
	}

	return ret, nil
}

func ensureOutputTensor(outputs map[string]*tf_proto.TensorProto, name string, dims int) error {
	output := outputs[name]
	if output == nil {
		return errors.Errorf("got output for '%s'", name)
	}

	if len(output.TensorShape.GetDim()) != dims {
		return errors.Errorf("'%s' has unexpected dims: %d, expected: %d",
			name, len(output.TensorShape.GetDim()), dims)
	}

	return nil
}
