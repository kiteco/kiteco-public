package livemetrics

var (
	// kitestatusAllowed keeps track of all the variables registered with
	// kitestatus pkg that are allowed into kite_status
	kitestatusAllowed = map[string]bool{
		"spyder_suboptimal_settings": true,
	}
)
