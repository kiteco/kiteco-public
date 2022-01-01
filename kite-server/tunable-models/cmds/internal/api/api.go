package api

// Model ...
type Model struct {
	Name   string
	Active bool
}

// ListModelResponse ...
type ListModelResponse struct {
	Status       string
	Models       []Model
	Repositories []string
}

// DeleteRequest ...
type DeleteRequest struct {
	Model      string
	Repository string
}

// TuneModelRequest ...
type TuneModelRequest struct {
	Repository string
	Swap       bool
}

// SwapModelRequest ...
type SwapModelRequest struct {
	SwapToModel string
}

// MessageResponse ...
type MessageResponse struct {
	Message string
}
