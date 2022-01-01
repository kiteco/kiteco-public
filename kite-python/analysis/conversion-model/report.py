import json
import string


def main():
    with open("report.txt", "r") as fp:
        template = string.Template(fp.read())

    with open("params.json", "r") as fp:
        params = json.load(fp)

    lines = [
        "parameter|value",
        "-|-",
    ]
    lines.extend(f"{param}|{value}" for param, value in params.items())
    table = "\n".join(lines)
    report = template.substitute(coefficients=table)

    with open("README.md", "w") as fp:
        fp.write(report)


if __name__ == "__main__":
    main()
