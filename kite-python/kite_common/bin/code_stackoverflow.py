import argparse

from kite.coderetrieval import stackoverflow


DELIM = "\n----------------------------------------------\n"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("full_stackoverflow_scrape_in_xml")
    parser.add_argument("tags", nargs='+')
    args = parser.parse_args()

    code_from_qns = stackoverflow.extract_code(args.full_stackoverflow_scrape_in_xml,
                                               stackoverflow.SO_QUESTION_TYPE,
                                               args.tags)
    code_from_ans = stackoverflow.extract_code(args.full_stackoverflow_scrape_in_xml,
                                               stackoverflow.SO_ANSWER_TYPE)

    with open("code_from_qns", "w") as fp:
        print(DELIM.join(code_from_qns), file=fp)

    with open("code_from_ans", "w") as fp:
        print(DELIM.join(code_from_ans), file=fp)

if __name__ == "__main__":
    main()
