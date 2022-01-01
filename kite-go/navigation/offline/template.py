import csv
import string


def main():
    with open("template.txt", "r") as fp:
        template = string.Template(fp.read())

    validation = csv_to_md("validation.csv")
    performance = csv_to_md("performance.csv")
    sub = template.substitute(validation=validation, performance=performance)

    with open("README.md", "w") as fp:
        fp.write(sub)


def csv_to_md(path):
    with open(path, "r") as fp:
        csvreader = csv.reader(fp)
        header, *data = list(csvreader)

    align = ["-"] + [":-:"] * (len(header) - 1)
    rows = [header, align] + data
    return "\n".join("|".join(row) for row in rows)


if __name__ == "__main__":
    main()
