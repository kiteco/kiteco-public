import csv
import datetime
import json
import string


def main():
    with open("template.txt", "r") as fp:
        template = string.Template(fp.read())

    report = template.substitute(
        date=str(datetime.datetime.now().date()),
        control_validation=format_validation("control"),
        treatment_validation=format_validation("treatment"),
        examples=format_examples(),
    )

    with open("README.md", "w") as fp:
        fp.write(report)


def format_validation(cohort):
    with open(f"{cohort}_validation.csv", "r") as fp:
        csvreader = csv.reader(fp)
        header, *data = csvreader

    lines = [
        "|".join(header),
        "|".join("-" if i == 0 else ":-:" for i in range(len(header)))
    ] + [
        "|".join(map(str, row)) for row in data
    ]
    return "\n".join(lines)


def format_examples():
    with open("control_examples.json", "r") as fp:
        control = json.load(fp)

    with open("treatment_examples.json", "r") as fp:
        treatment = json.load(fp)

    assert len(control) == len(treatment)
    examples = [
        format_example(i, c, t)
        for i, (c, t) in enumerate(zip(control, treatment))
    ]
    return "\n\n".join(examples)


def format_example(i, control, treatment):
    assert control["Input"] == treatment["Input"]

    current_path = control["Input"]["CurrentPath"]
    related_path = control["Input"]["RelatedPath"]
    github_master = "https://github.com/kiteco/kiteco/blob/master"
    current_url = f"{github_master}/{current_path}"
    related_url = f"{github_master}/{related_path}"
    control_path_rank, control_keywords = control["Result"].values()
    treatment_path_rank, treatment_keywords = treatment["Result"].values()

    lines = [
        "",
        f"### Example {i+1}",
        "",
        "#### Input",
        f"- Current path: [`{current_path}`]({current_url})",
        f"- Related path: [`{related_path}`]({related_url})",
        "",
        "#### Related path rank",
        f"- Control: `{control_path_rank}`",
        f"- Treatment: `{treatment_path_rank}`",
        "",
        "#### Related path keywords",
        "|Control|Treatment|",
        "|-|-|",
    ] + [
        f"|`{c_keyword}`|`{t_keyword}`|"
        for c_keyword, t_keyword in zip(control_keywords, treatment_keywords)
    ]
    return "\n".join(lines)


if __name__ == "__main__":
    main()
