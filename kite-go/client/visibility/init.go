package visibility

var c *checker

func init() {
	c = newChecker()
}

// Listen registers a callback that will receive visibility notifications. Any
// previous listener registered with the same key will be removed.
func Listen(key string, l Listener) {
	c.Listen(key, l)
}

// RecentlyVisible returns whether the sidebar has been visible in the last 10 seconds
func RecentlyVisible() bool {
	return c.Recently
}

// Clear removes all listeners of visibility notifications
func Clear() {
	c.clear()
}
