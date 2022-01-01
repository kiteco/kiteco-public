class IdentFeaturizer(object):
    def features(self, data):
        return [float(t) for t in data.split()] 
