import re

STRONG_MATCH = "strong_match"
WEAK_MATCH = "weak_match"
NO_MATCH = "no_match"
WITHIN_EXAMPLE_DELIM = "----------------------"
EXAMPLE_DELIM = "----------------------------------------------"

STRONG_STACKTRACE_START_PATTERNS = [r"goroutine \d+ \[.+\]:"]
WEAK_STACKTRACE_PATTERNS_SET = [r"\.[a-z]+:[0-9]+\s?", r"\S+\(.*\)"]


def match_known_weak_stacktrace_patterns(line):
    return any([re.search(weak_pattern, line, re.I)
                for weak_pattern in WEAK_STACKTRACE_PATTERNS_SET])

def matches_known_stacktrace_patterns(line_one, line_two):
    for strong_match_pattern in STRONG_STACKTRACE_START_PATTERNS:
        if re.search(strong_match_pattern, line_one, re.I):
            return STRONG_MATCH

    if (match_known_weak_stacktrace_patterns(line_one)
                and match_known_weak_stacktrace_patterns(line_two)):
        return WEAK_MATCH

    return NO_MATCH

def extract_stacktraces_from_string(s):
    if s is None:
        return []

    stacktraces = []
    lines = s.split("\n")

    stacktrace = ""
    within_strong_trace = False
    within_weak_trace = False
    for idx in range(len(lines) - 1):
        curr_line = lines[idx].strip()
        next_line = lines[idx + 1].strip()

        strength = matches_known_stacktrace_patterns(curr_line, next_line)
        if within_strong_trace or within_weak_trace:
            was_and_no_longer_weak = (within_weak_trace and (strength == NO_MATCH))
            if len(curr_line) == 0 or was_and_no_longer_weak:
                within_strong_trace = False
                within_weak_trace = False
                if was_and_no_longer_weak:
                    stacktrace += curr_line
                stacktraces.append(stacktrace.strip())
                stacktrace = ""
            else:
                stacktrace += curr_line + "\n"
        else:
            if strength == STRONG_MATCH:
                within_strong_trace = True
                stacktrace += curr_line + "\n"
            elif strength == WEAK_MATCH:
                within_weak_trace = True
                stacktrace += curr_line + "\n"

    if len(stacktrace) > 0:
        last_line = lines[-1].strip()
        if match_known_weak_stacktrace_patterns(last_line):
            stacktrace += last_line
        stacktraces.append(stacktrace.strip())

    return stacktraces

def remove_formatting(text):
    return "\n".join([line.strip() for line in text.split("\n")])

def results_to_string_helper(result):
    text = result[0]
    stacktraces = result[1]
    ground_truth = result[2] if len(result) == 3 else None

    output = ""

    text = text.strip()
    if ground_truth is None:
        output += (text + "\n" + WITHIN_EXAMPLE_DELIM + "\n")
        if len(errors) > 0:
            output += (
                "\n\n".join(stacktraces) + "\n" + EXAMPLE_DELIM + "\n")
        else:
            output += (
                "No stacktraces extracted\n" + EXAMPLE_DELIM + "\n")
    else:
        concat_stacktraces = "\n\n".join(stacktraces)
        ground_truth = ground_truth.strip()
        output += (text + "\n" + WITHIN_EXAMPLE_DELIM + "\n")
        output += ("Ground truth:\n" + ground_truth + "\n" + WITHIN_EXAMPLE_DELIM + "\n")
        output += ("Got:\n" + concat_stacktraces + "\n")
        if remove_formatting(ground_truth) == concat_stacktraces:
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

def extract_stacktraces_from_file(f):
    extraction_results = []
    with open(f) as fo:
        t = fo.read()
        code = t.split(EXAMPLE_DELIM)

        for snippet in code:
            snippet = snippet.strip()
            if len(snippet) > 0:
                stacktraces = extract_stacktraces_from_string(snippet)
                extraction_results.append((snippet, stacktraces))

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

                stacktraces = extract_stacktraces_from_string(code)
                eval_results.append((code, stacktraces, ground))

    return eval_results
