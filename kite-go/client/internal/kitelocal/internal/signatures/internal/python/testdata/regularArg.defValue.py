def myFunc(first="default first", second="default second", **kwargs):
    print(first + " |  " + second)

myFunc(kw1="kw1 value", kw2="kw2 value")
myFunc(first="my 1st", second="my 2nd"<caret>)
