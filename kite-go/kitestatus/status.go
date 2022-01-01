package kitestatus

// Get returns all the metrics kitestatus knows about
func Get() map[string]interface{} {
	collected := make(map[string]interface{})
	metrics.Range(func(key, value interface{}) bool {
		switch t := value.(type) {
		case Metric:
			for k, v := range t.Value() {
				collected[k] = v
			}
		}
		return true
	})
	return collected
}

// Reset resets all the metrics
func Reset() {
	metrics.Range(func(key, value interface{}) bool {
		switch t := value.(type) {
		case Metric:
			t.Reset()
		}
		return true
	})

}
