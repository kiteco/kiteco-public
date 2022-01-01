package main

import "net/http"

func (a *app) handleRoot(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	sortParam := params.Get("sort")

	ls := newListingSet(a.collection, "")

	byProvider := ls.ProviderBreakdowns()
	bySymbol := ls.SymbolBreakdowns(sortParam)

	err := a.templates.Render(w, "root.html", map[string]interface{}{
		"Count":      len(ls.Listings),
		"Path":       a.collection.Path,
		"BySymbol":   bySymbol,
		"ByProvider": byProvider,
		"Providers":  ls.Providers,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
