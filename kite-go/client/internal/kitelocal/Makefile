distranking-bindata:
	go build github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/cmds/distribution-ranking
	./distribution-ranking -output=distscores.json
	rm distribution-ranking
	go generate github.com/kiteco/kiteco/kite-go/client/distranking
	rm distscores.json
