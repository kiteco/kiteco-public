package main

import (
	"flag"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang"
)

// This is a simple script for training language detectors.
// usage: ./train -dataDir=<dir that contains training data> -modelFile=<output model file>
// Get the training data from s3://kite-data/training-data/language_detection.

func main() {
	var dataDir, modelFile string

	flag.StringVar(&dataDir, "dataDir", "", "Directory that contains training data for a programming language")
	flag.StringVar(&modelFile, "modelFile", "", "Output model filename")
	flag.Parse()

	detector := lang.NewLanguageDetector()
	if _, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Error: %s does not exist. Language detector is not trained.", dataDir)
		} else {
			log.Println("Error:", err)
		}
	} else {
		log.Printf("Training a language detector with data in %s", dataDir)
		detector.Model.TrainFromData(dataDir)
		detector.Model.SaveModel(modelFile)
		log.Printf("Saved model in %s\n", modelFile)
	}
}
