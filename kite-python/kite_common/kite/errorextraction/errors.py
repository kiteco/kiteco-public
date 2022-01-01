import re


STRONG_MATCH = "strong"
WEAK_MATCH = "weak"
NO_MATCH = "nomatch"
WITHIN_EXAMPLE_DELIM = "----------------------"
EXAMPLE_DELIM = "----------------------------------------------"

def matches_known_error_patterns(line):
    strong_error_patterns = [
        r"error:[^=]",
        r"fatal:[^=]",
        r"panic:[^=]",
        r"runtime:[^=]",
        r"http:[^=/]",
        r"\.go:[^=]+:",
        r"go-app-builder:[^=]",
        r"goinstall:[^=]"]
    weak_error_patterns = ["error", "fatal", "panic", "runtime", "http"]

    for pattern in strong_error_patterns:
        if re.search(pattern, line, re.I):
            return STRONG_MATCH

    num_weak_patterns_matched = 0
    for pattern in weak_error_patterns:
        occurences = re.finditer(pattern, line)
        indices = [occurence.start() for occurence in occurences]
        if len(indices) > 0:
            num_weak_patterns_matched += len(indices)

    if num_weak_patterns_matched >= 2:
        return WEAK_MATCH

    return NO_MATCH


def is_comment(line):
    return line.strip().startswith("//")


def is_go_code(line):
    return any((x in line) for x in [":=", "func", "if", "for", "fmt"])


def extract_errors_from_string(s):
    if s is None:
        return []

    errors = []
    lines = s.split("\n")
    for line in lines:
        line = line.strip()
        strength = matches_known_error_patterns(line)
        if strength == STRONG_MATCH or (strength == WEAK_MATCH and \
                not is_comment(line) and \
                not is_go_code(line) and \
                len(line.split(" ")) > 3):
            errors.append(line)
    return errors


def results_to_string_helper(result):
    text = result[0]
    errors = result[1]
    ground_truth = result[2] if len(result) == 3 else None

    output = ""

    text = text.strip()
    if ground_truth is None:
        output += (text + "\n" + WITHIN_EXAMPLE_DELIM + "\n")
        if len(errors) > 0:
            output += (
                "\n".join(errors) + "\n" + EXAMPLE_DELIM + "\n")
        else:
            output += (
                "No errors extracted\n" + EXAMPLE_DELIM + "\n")
    else:
        concat_errors = "\n".join(errors)
        ground_truth = ground_truth.strip()
        output += (text + "\n" + WITHIN_EXAMPLE_DELIM + "\n")
        output += ("Ground truth:\n" + ground_truth + "\n" + WITHIN_EXAMPLE_DELIM + "\n")
        output += ("Got:\n" + concat_errors + "\n")
        if ground_truth == concat_errors:
            output += ("$$$$$$$ SUCCESS $$$$$$$\n")
        else:
            output += ("^^^^^^^ FAIL ^^^^^^^\n")

        output += (EXAMPLE_DELIM + "\n")

    return output

def results_to_string(extraction_or_eval_results):
    output = ""
    for result in extraction_or_eval_results:
        output += results_to_string_helper(result)

    return output

def extract_errors_from_file(f):
    extraction_results = []
    with open(f) as fo:
        t = fo.read()
        code = t.split(EXAMPLE_DELIM)

        for snippet in code:
            snippet = snippet.strip()
            if len(snippet) > 0:
                errors = extract_errors_from_string(snippet)
                extraction_results.append((snippet, errors))

    return extraction_results

def evaluate_ground_truth(ground_truth_file):
    eval_results = []
    with open(ground_truth_file) as fo:
        t = fo.read()
        test_examples = t.split(EXAMPLE_DELIM)

        for example in test_examples:
            example = example.strip()
            if len(example) > 0:
                code_and_ground = example.split(WITHIN_EXAMPLE_DELIM)
                code = code_and_ground[0].strip()
                ground = ""
                if len(code_and_ground) == 2:
                    ground = code_and_ground[1].strip()

                errors = extract_errors_from_string(code)
                eval_results.append((code, errors, ground))

    return eval_results
