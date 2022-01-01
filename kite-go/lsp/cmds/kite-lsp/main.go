package main

import (
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lsp"

	"github.com/kiteco/kiteco/kite-go/lsp/jsonrpc2"
	"github.com/kiteco/kiteco/kite-go/lsp/process"
)

func main() {
	log.Println("Kite-LSP Starting...")

	startKite()

	stdio := jsonrpc2.NewReaderWriterConnection(os.Stdin, os.Stdout)

	var lspServer *lsp.Server
	lspServer = lsp.New()

	rpcConn := jsonrpc2.NewRPCConnection(stdio, lspServer)

	err := rpcConn.Run()
	if err != nil {
		log.Println(err)
		log.Println("RPC connection closed.")
	}
}

func startKite() {
	// Attempt to start Kite if it's not running.
	isRunning, err := process.IsRunning(process.Name)
	if err != nil {
		log.Println("Could not check if Kite is running. Continuing initialization...")
		return
	}
	if !isRunning {
		log.Println("Kite not running! Attempting to start...")
		err = process.Start()
		if err != nil {
			log.Println("Could not autostart Kite:", err)
		}
	}
}
