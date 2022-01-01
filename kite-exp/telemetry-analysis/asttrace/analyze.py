#!/usr/bin/env python3
from collections import Counter
import fileinput
import json


class Stats:
    def __init__(self):
        self.got_completions = 0
        self.handled_no_completions = 0

        self.no_ast = 0
        self.end_of_node = 0
        self.type_counts = Counter()

    def handle(self, blob):
        if blob['num_completions'] > 0:
            self.got_completions += 1
            return

        if blob['handled']:
            self.handled_no_completions += 1
            return

        trace = blob['trace']['Trace']
        if not trace['Nodes']:
            self.no_ast += 1
            return

        if trace['Nodes'][-1]['End'] == trace['Cursor']:
            self.end_of_node += 1
            return

        self.type_counts[trace['Nodes'][-1]['Type']] += 1

    def print(self):
        print('handled, no completions:', self.handled_no_completions)
        print('no AST:', self.no_ast)
        print('end of node:', self.end_of_node)
        print('middle of node:', sum(self.type_counts.values()))
        for ty, cnt in self.type_counts.most_common(10):
            print('  {}: {}'.format(ty, cnt))


def main():
    s = Stats()
    for line in fileinput.input():
        s.handle(json.loads(line))
    s.print()


if __name__ == '__main__':
    main()
