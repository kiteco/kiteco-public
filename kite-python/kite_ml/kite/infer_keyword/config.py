class Config(object):
    def __init__(self):
        self.validate_input = True
        # number of previous tokens to use for classification
        self.lookback = 5
        # number of training epochs
        self.n_epochs = 5
        # batch size for each training batch
        self.batch_size = 128
