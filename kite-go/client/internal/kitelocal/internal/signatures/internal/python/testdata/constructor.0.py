class Dog:
    def __init__(self, name, *nickName, **kwargs):
        self.name = name

Dog("name<caret>", "dog", "brutus", "roger", kw1="first")