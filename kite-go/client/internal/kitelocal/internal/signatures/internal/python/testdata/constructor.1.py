class Dog:
    def __init__(self, name, *nickName, **kwargs):
        self.name = name

Dog("name", "dog", "brutus", "roger<caret>", kw1="first")