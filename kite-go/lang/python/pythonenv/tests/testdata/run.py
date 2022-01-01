from server import Server

def start(port):
    def newServer(port):
        s = Server(port)
        return s
    server = newServer(port)
    server.start()

if __name__ == '__main__':
    start(":9090")