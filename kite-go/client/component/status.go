package component

// IsLoadeder allows asking if the resources (model) for a filetype is loaded and initialized
type IsLoadeder interface {
	IsLoaded(fext string) bool
}

// Validator checks the filepath for errors
type Validator interface {
	Validate(path string) error
}

// StatusManager provides a setter
type StatusManager interface {
	SetModels(IsLoadeder)
	SetNav(Validator)
}
