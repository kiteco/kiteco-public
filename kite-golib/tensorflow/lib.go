package tensorflow

import (
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-golib/applesilicon"
)

func init() {
	// We do not support tensorflow models in Apple Silicon
	if applesilicon.Detected {
		return
	}

	start := time.Now()
	defer func() {
		log.Printf("loading tensorflow took %s", time.Now().Sub(start))
	}()

	log.Print("loading tensorflow dynamically...")
	if err := loadTensorflow(); err != nil {
		panic(err)
	}
}
