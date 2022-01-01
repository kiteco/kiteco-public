package sysidle

var c *checker

func init() {
	c = newChecker()
}

// Listen registers a callback that will receive system idle notifications. Any
// previous listener registered with the same key will be removed.
func Listen(key string, l Listener) {
	c.Listen(key, l)
}

// Clear removes all listeners of system idle notifications
func Clear() {
	c.clear()
}
