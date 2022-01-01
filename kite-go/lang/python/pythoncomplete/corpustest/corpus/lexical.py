def test_lexical_filter_matplotlib():
    import numpy as np
    import matplotlib.pyplot as plt

    x = np.linspace(0, 1)
    y = np.sin(x)

    fig, ax = plt.subplots(figsize(5, 5))
    '''TEST
    ax.sca$
    @. `scatter(x, y)`
    status: fail
    '''

def test_no_exact_match():
    import pandas
    url = "http.com"
    df = pandas.read_csv(url)
    '''TEST
    df.asdfas$
    @! asdfas asdfas
    status: ok
    '''

def test_lexical_display_requests():
    import requests

    '''TEST
    resp = requ$
    @. requests.get
    @! requests.get() requests.get(…)
    status: ok
    '''

def test_lexical_dedupe_snippet():
    import matplotlib.pyplot as plt
    import pandas as pd

    df = pd.read_csv("data.csv")
    plt.plot(df["date"], df["revenue"])
    filename = "revenue.png"
    '''TEST
    plt.s$
    @. savefig(filename) savefig(filename) ★
    status: fail
    '''

def test_lexical_dedupe_keyword_without_space():
    '''TEST
    def fn(): r$
    @. return return keyword
    status: ok
    '''

def test_lexical_dedupe_keyword_with_space():
    '''TEST
    def fn(): y$
    @. `yield ` yield keyword
    status: ok
    '''

def test_lexical_nonempty():
    '''TEST
    def fn():
        y$
    @! y
    status: ok
    '''

def test_single_token_midline():
    import numpy as np

    start = 1
    stop = 10
    num = 30

    '''TEST
    x = np.linspace(start=s$, stop=stop, num=num)
    @. start start
    @! `start, stop=stop,`
    @! `start, stop=stop`
    @! `start, stop=`
    @! `start, stop`
    @! start,
    status: ok
    '''
