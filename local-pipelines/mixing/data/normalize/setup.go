package normalize

import (
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// Setup resource manager and models
func Setup() (pythonresource.Manager, *pythonmodels.Models, *lexicalmodels.Models, error) {
	resourceManager, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		return nil, nil, nil, err
	}
	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	if err != nil {
		return nil, nil, nil, err
	}
	lexicalmodels, err := lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
	if err != nil {
		return nil, nil, nil, err
	}
	return resourceManager, models, lexicalmodels, nil
}
