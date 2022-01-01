package main

import (
	"bufio"
	"io"
	"os"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"

	"github.com/alexflint/go-arg"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		OutputPath string
		Language   string
	}{
		OutputPath: "",
		Language:   "javascript",
	}
	arg.MustParse(&args)

	fail(datadeps.Enable())
	fileutil.SetLocalOnly()

	l := lexicalv0.MustLangGroupFromName(args.Language)
	modelOptions, err := lexicalmodels.GetDefaultModelOptions(l)
	fail(err)
	modelOutput := "lexical_model.frozen.pb"
	modelPath := fileutil.Join(modelOptions.ModelPath, modelOutput)
	configOutput := "config.json"
	configPath := fileutil.Join(modelOptions.ModelPath, configOutput)
	vocabOutput := "ident-vocab-entries.bpe"
	vocabPath := fileutil.Join(modelOptions.ModelPath, vocabOutput)
	if args.OutputPath != "" {
		modelOutput = fileutil.Join(args.OutputPath, modelOutput)
		configOutput = fileutil.Join(args.OutputPath, configOutput)
		vocabOutput = fileutil.Join(args.OutputPath, vocabOutput)
	}
	fail(extractFile(modelPath, modelOutput))
	fail(extractFile(configPath, configOutput))
	fail(extractFile(vocabPath, vocabOutput))
}

func extractFile(inputPath string, outputPath string) error {
	reader, err := fileutil.NewCachedReader(inputPath)
	if err != nil {
		return err
	}
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(output)
	nByte, err := io.Copy(w, reader)
	if err != nil {
		return err
	}
	if nByte == 0 {
		return errors.Errorf("No byte have been written when extracting the file %s (is it empty?)", inputPath)
	}
	if err = w.Flush(); err != nil {
		return err
	}
	if err = reader.Close(); err != nil {
		return err
	}
	if err = output.Close(); err != nil {
		return err
	}
	return nil
}
