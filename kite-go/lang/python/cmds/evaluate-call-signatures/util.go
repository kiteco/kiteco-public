package main

import "fmt"

func ftoa(n float64) string {
	return fmt.Sprintf("%.2f", n)
}

func nodeURL(anyname string) string {
	if anyname == "" {
		return ""
	}
	return fmt.Sprintf("http://graph.kite.com/node/%s", anyname)
}
