package render

import (
	"regexp"

	"github.com/kiteco/kiteco/kite-answers/go/execution"
)

var commentRe = regexp.MustCompile("#\\s*(.+)")

func annotateBlocks(blocks []execution.Block, lang string) []CodeBlockItem {
	var processedBlocks []CodeBlockItem
	var annotationBlock []string

	for _, block := range blocks {
		if block.CodeLine != nil {
			codeLine := *block.CodeLine
			// Check if block.CodeLine is a comment line and grab content without leading and trailing whitespace.
			match := commentRe.FindStringSubmatch(codeLine)
			if match != nil {
				annotationBlock = append(annotationBlock, match[1])
				continue
			}
		}
		// If block's CodeLine is not a comment, or encapsulates Output, attach annotation block.
		processedBlocks = append(processedBlocks, CodeBlockItem{Block: block, Lang: lang, AnnotationBlock: annotationBlock})
		annotationBlock = nil
	}

	// If the annotation array is non-empty at this point, roll in with last line of code.
	if len(annotationBlock) > 0 {
		if len(processedBlocks) > 0 {
			lastLine := &processedBlocks[len(processedBlocks)-1]
			lastLine.AnnotationBlock = append(lastLine.AnnotationBlock, annotationBlock...)
		} else {
			// If annotations don't have line of code to attach, render error.
			var errorBlock CodeBlockItem
			errorBlock.Output = &execution.Output{
				Type:  "text",
				Title: "!! Invalid Code Block",
				Data:  "No executable code in code block",
			}
			processedBlocks = append(processedBlocks, errorBlock)
		}
		annotationBlock = nil
	}
	return processedBlocks
}
