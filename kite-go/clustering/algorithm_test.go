package clustering

import (
	"testing"
)

// TestNewKMeans test if NewKMeans handles invalid input properly.
func TestNewKMeans(t *testing.T) {
	km, err := NewKMeans(3)
	if km.K != 3 {
		t.Errorf("Expected km.K to be 3, but got %d.\n", km.K)
	}

	km, err = NewKMeans(-1)
	if err == nil {
		t.Errorf("Expected NewKMeans(-1) to return an error.")
	}

	if km != nil {
		t.Errorf("Expected NewKMeans(-1) to return nil.")
	}
}
