class Constants(object):

    # Number of letter in the alphabet (for python keyword, ascii lowercase)
    LETTER_COUNT = 26
    # Number of prefixes class
    # Letter count + 1 for nothing (space) + 1 for anything else
    N_PREFIXES = LETTER_COUNT + 2
    # number of token classes
    # refer to github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner/token.go
    N_TOKENS = 104
    # number of keyword classes
    # refer to github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword/mappings.go
    # 31 instead of 30 as it's 1-indexed and the validator is considering it 0-indexed
    N_KEYWORDS = 30
    # number AST node classes
    # refer to github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword/mappings.go
    N_NODES = 64
    # number of relative indent classes
    # refer to github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword/features.go
    N_REL_INDENT = 3

    def __init__(self):
        raise RuntimeError('namespace class only')
