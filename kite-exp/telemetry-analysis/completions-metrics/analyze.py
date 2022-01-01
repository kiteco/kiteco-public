import argparse
from collections import defaultdict
from datetime import datetime
import json


class Stats:
    def __init__(self):
        self.users = set()
        self.aggregates = defaultdict(lambda: defaultdict(lambda: 0))

    def handle(self, blob):
        # add user_id to set
        self.users.add(blob['user_id'])

        for breakdown in ["selected", "selected_2", "completed", "at_least_one_shown"]:
            agg = self.aggregates[breakdown]

            bd = blob[f"completions_{breakdown}_by_source"]
            if not bd:
                continue

            for typ in ["attribute_model", "traditional", "call_model", "keyword_model", "expr_model"]:
                count = bd.get(typ, 0)
                agg[typ] += count

                agg["total"] += count
                if typ in ("call_model", "expr_model"):
                    agg["mtac"] += count

    def for_json(self):
        return {
            'num_users': len(self.users),
            'aggregates': self.aggregates,
        }


class ShardedStats:
    def __init__(self):
        self.by_date = defaultdict(Stats)

    def handle(self, blob):
        dt = datetime.strptime(blob['timestamp'], "%Y-%m-%dT%H:%M:%S.%f%z")
        self.by_date[dt.date()].handle(blob)

    def for_json(self):
        return {str(k): v.for_json() for k, v in self.by_date.items()}


def main(fname):
    s = ShardedStats()
    with open(fname) as f:
        for line in f:
            blob = json.loads(line)
            s.handle(blob)

    print(json.dumps(s.for_json(), indent=2, sort_keys=True))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--logs', type=str, required=True)
    args = parser.parse_args()
    main(args.logs)
