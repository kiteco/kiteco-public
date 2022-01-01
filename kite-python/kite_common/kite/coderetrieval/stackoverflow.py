import sys
import collections
from lxml import etree
import pyquery


SO_QUESTION_TYPE = '1'
SO_ANSWER_TYPE = '2'

processed_line_count = 0
questions = dict()


CodeSnippet = collections.namedtuple('CodeSnippet',
    ['question_id', 'answer_id', 'view_count', 'question_score', 'answer_score'])


def extract_code(f, post_type, tags=None):
    context = etree.iterparse(f, tag='row')
    code = None
    if post_type == SO_QUESTION_TYPE:
        code = fast_iter(context, code_from_stackoverflow_helper,
                         SO_QUESTION_TYPE, tags)
    elif post_type == SO_ANSWER_TYPE:
        code = fast_iter(context, code_from_stackoverflow_helper,
                         SO_ANSWER_TYPE)

    return code


def count_processed_line():
    global processed_line_count
    processed_line_count += 1
    if processed_line_count % 1000000 == 0:
        print(str(processed_line_count), file=sys.stderr)


def fast_iter(context, func, post_type, tags=None):
    # Modified from following link:
    # http://www.ibm.com/developerworks/xml/library/x-hiperfparse/
    # Author: Liza Daly
    code_snippets = []

    for event, elem in context:
        code = func(elem, post_type, tags)
        code_snippets.extend(code)
        elem.clear()
        while elem.getprevious() is not None:
            del elem.getparent()[0]
    del context

    return code_snippets


def code_from_stackoverflow_helper(elem, post_type, tags):
    count_processed_line()

    code_snippets = []

    if elem.attrib['PostTypeId'] == post_type:
        if post_type == SO_QUESTION_TYPE:
            elem_tags = elem.attrib['Tags']
            elem_tags = elem_tags.replace("><", ",")
            elem_tags = elem_tags.replace("<", ",")
            elem_tags = elem_tags.replace(">", ",")
            elem_tags = elem_tags.split(",")
            elem_tags = [tag for tag in elem_tags if len(tag) > 0]

            tag_found = any((tag in elem_tags) for tag in tags)
            if tag_found:
                eid = int(elem.attrib['Id'])
                questions[eid] = (
                    elem.attrib['ViewCount'],
                    elem.attrib['Score'],
                    elem.attrib.get('AcceptedAnswerId', None))

                p = pyquery.pyquery.PyQuery(elem.attrib['Body'])
                for code in p('code'):
                    if code.text:
                        try:
                            t = code.text.encode('ascii', 'replace')
                            t = t.decode("utf-8")
                            code_snippets.append(t)
                        except UnicodeEncodeError:
                            pass
        elif post_type == SO_ANSWER_TYPE:
            parent_id = int(elem.attrib['ParentId'])
            if parent_id not in questions:
                return []
            print(elem.attrib.keys())
            exit(0)
            view_count, question_score, accepted_id = questions[parent_id]
            p = pyquery.pyquery.PyQuery(elem.attrib['Body'])
            for code in p('code'):
                if code.text:
                    try:
                        t = code.text.encode('ascii', 'replace')
                        t = t.decode("utf-8")
                        code_snippets.append(t)
                    except UnicodeEncodeError:
                        pass

    return code_snippets
