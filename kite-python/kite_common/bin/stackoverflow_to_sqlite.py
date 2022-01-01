import sys
import collections
from lxml import etree
import pyquery

import sqlite3

def tags(s):
    """Split '<foo><bar><baz>' into ['foo', 'bar', 'baz']."""
    return ' '.join(s[1:-1].split('><'))

XmlFields = [
    ('Id', int),
    ('PostTypeId', int),
    ('AcceptedAnswerId', int),
    ('CreationDate', str),
    ('Score' int),
    ('ViewCount', int),
    ('Body' str),
    ('OwnerUserId', int),
    ('LastActivityDate', str),
    ('Title', str),
    ('Tags', tags),
    ('AnswerCount', int),
    ('CommentCount', int),
    ('FavoriteCount', int)
    ]


def fast_iter(context, func, limit=None):
    # Modified from following link:
    # http://www.ibm.com/developerworks/xml/library/x-hiperfparse/
    # Author: Liza Daly
    n = 0
    for event, elem in context:
        n += 1
        if n % 100000 == 0:
            print(n)
        func(elem)
        elem.clear()
        while elem.getprevious() is not None:
            del elem.getparent()[0]
        if limit is not None and n >= limit:
            return


def extract_code_blocks(s):
    endpos = -1
    while True:
        beginpos = s.find('<code>', endpos+1)
        if beginpos == -1:
            break
        endpos = s.find('</code>', beginpos)
        if endpos == -1:
            break
        yield s[beginpos+len('<code>'):endpos]


def main():
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('stackoverflow_dump')
    parser.add_argument('database')
    args = parser.parse_args()

    # Open database
    conn = sqlite3.connect(args.database)
    c = conn.cursor()
    c.execute("""CREATE TABLE posts (
             id INTEGER PRIMARY KEY,
             post_type INTEGER,
             accepted_answer_id INTEGER,
             creation_date TIMESTAMP,
             score INTEGER,
             view_count INTEGER,
             body TEXT,
             owner_user_id INTEGER,
             last_activity_date timestamp,
             title TEXT,
             tags TEXT,
             answer_count INTEGER,
             comment_count INTEGER,
             favourite_count INTEGER)""")

    c.execute("""CREATE TABLE code_blocks (
             post_id INTEGER,
             offset INTEGER,
             content TEXT
             )""")

    conn.commit()
    c.close()

    posts = []
    code_blocks = []

    def flush():
        c = conn.cursor()
        print('Inserting %d posts and %d code blocks...' % (len(posts), len(code_blocks)))
        if len(posts) > 0:
            c.executemany('INSERT INTO posts VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)', posts)
            del posts[:]
        if len(code_blocks) > 0:
            c.executemany('INSERT INTO code_blocks VALUES (?,?,?)', code_blocks)
            del code_blocks[:]
        conn.commit()
        c.close()

    def process_node(elem):
        item = []
        for tag, cast in XmlFields:
            x = elem.attrib.get(tag, None)
            if x is not None:
                x = cast(x)
            item.append(x)
        posts.append(item)

        if 'Body' in elem.attrib and 'Id' in elem.attrib:
            for i, content in enumerate(extract_code_blocks(elem.attrib['Body'])):
                code_blocks.append([int(elem.attrib['Id']), i, content])

        if len(posts) + len(code_blocks) > 1000000:
            flush()

    # Parse the xml file
    print('Traversing XML...')
    context = etree.iterparse(args.stackoverflow_dump)
    fast_iter(context, process_node)

    # Flush
    flush()

if __name__ == '__main__':
    main()
