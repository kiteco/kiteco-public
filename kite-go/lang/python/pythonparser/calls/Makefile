PARSER_DIR := internal/pigeon

PARSER_GRAMMAR := $(PARSER_DIR)/parser.peg
GENERATED_PARSER := $(PARSER_DIR)/gen_parser.go

DEBUG ?= 0
PIGEON_FLAGS =

# Use: $ DEBUG=1 make
# to generate the parser without any optimization.
ifneq ($(DEBUG),1)
	PIGEON_FLAGS += --alternate-entrypoints GrammarArgsOnly,GrammarStmt,TestID,TestAttr,TestNumber,TestStrings,TestList,TestDictSet,TestTuple --optimize-parser --optimize-grammar --optimize-basic-latin
endif

.PHONY: all
all: $(GENERATED_PARSER)

$(GENERATED_PARSER): $(PARSER_GRAMMAR)
	pigeon $(PIGEON_FLAGS) -o $@ $^

.PHONY: clean
clean:
	rm -f $(GENERATED_PARSER)

