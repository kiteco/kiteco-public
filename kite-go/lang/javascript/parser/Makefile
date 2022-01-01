PARSER_DIR = internal/pigeon


PARSER_GRAMMAR = $(PARSER_DIR)/javascript.peg
PARSER = $(PARSER_DIR)/javascript.go


default: all

all: parser

parser: $(PARSER)

$(PARSER):
	pigeon --optimize-parser --optimize-grammar --optimize-basic-latin $(PARSER_GRAMMAR) > $(PARSER)
	goimports -w $(PARSER_DIR)/*.go

clean:
	rm $(PARSER)