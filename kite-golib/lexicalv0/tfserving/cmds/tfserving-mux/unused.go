package main

import (
	"context"

	serving_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/serving"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// -- ignore these, they are here to satisfying the PredictionService interface

func (*server) Classify(context.Context, *serving_proto.ClassificationRequest) (*serving_proto.ClassificationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Classify not implemented")
}
func (*server) Regress(context.Context, *serving_proto.RegressionRequest) (*serving_proto.RegressionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Regress not implemented")
}
func (*server) MultiInference(context.Context, *serving_proto.MultiInferenceRequest) (*serving_proto.MultiInferenceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MultiInference not implemented")
}
func (*server) GetModelMetadata(context.Context, *serving_proto.GetModelMetadataRequest) (*serving_proto.GetModelMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetModelMetadata not implemented")
}
