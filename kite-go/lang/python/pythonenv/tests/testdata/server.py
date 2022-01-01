class Server():
    def __init__(self, port):
        self.port = port
        self.running = False
    
    def start(self):
        self.running = True

