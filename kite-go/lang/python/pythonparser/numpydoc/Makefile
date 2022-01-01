PARSER_DIR := internal/pigeon
INLINE_PARSER_DIR := internal/inline

PARSER_GRAMMAR := $(PARSER_DIR)/parser.peg
GENERATED_PARSER := $(PARSER_DIR)/gen_parser.go
INLINE_PARSER_GRAMMAR := $(INLINE_PARSER_DIR)/parser.peg
INLINE_GENERATED_PARSER := $(INLINE_PARSER_DIR)/gen_parser.go

DEBUG ?= 0
PIGEON_FLAGS =

# Use: $ DEBUG=1 make
# to generate the parser without any optimization.
ifneq ($(DEBUG),1)
	PIGEON_FLAGS += --optimize-parser --optimize-basic-latin --optimize-grammar
endif

.PHONY: all
all: $(GENERATED_PARSER) $(INLINE_GENERATED_PARSER)

$(GENERATED_PARSER): $(PARSER_GRAMMAR)
	pigeon $(PIGEON_FLAGS) -o $@ $^

$(INLINE_GENERATED_PARSER): $(INLINE_PARSER_GRAMMAR)
	pigeon $(PIGEON_FLAGS) -o $@ $^

.PHONY: clean
clean:
	rm -f $(GENERATED_PARSER)
	rm -f $(INLINE_GENERATED_PARSER)
