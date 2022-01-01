import json, collections, random, functools


def compose(*functions):
    return functools.reduce(lambda f, g: lambda x: f(g(x)), functions, lambda x: x)


def dist(s):
    return s.split(':')[0]


def path(s):
    return s.split(':')[1]


def pathlen(p):
    return p.count('.') + 1


def main():
    with open('docs.json') as f:
        data = json.load(f)

    def process((doc, freq)):
        pathlens = map(compose(pathlen, path), data[doc])
        minlen = min(pathlens)
        num_minlen = pathlens.count(minlen)
        return {
            'doc': doc,
            'freq': freq,
            'sample': random.sample(data[doc], 2),
            'num_dists': len(set(map(dist, data[doc]))),
            'minlen': minlen,
            'num_minlen': num_minlen,
        }

    counts = collections.Counter({k: len(v) for k, v in data.items() if len(v) > 1})
    processed = map(process, counts.most_common())

    candidates = [v['freq'] for v in processed if v['num_dists'] == 1 and v['num_minlen'] == 1]
    # print(json.dumps(candidates, indent=2))
    print(sum(candidates) - len(candidates))


if __name__ == '__main__':
    main()
