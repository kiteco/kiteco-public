package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

var debug = false

// runCmds runs a series of commands, returning the combined output of the last command or an error
// if any of the commands returns a non-zero status code.
func runCmds(cmds []*exec.Cmd) (string, error) {
	var lastOutput string

	for _, cmd := range cmds {
		if debug {
			log.Printf(">> %s", strings.Join(cmd.Args, " "))
		}
		combined, err := cmd.CombinedOutput()
		lastOutput = string(combined)
		if debug && len(lastOutput) > 0 {
			log.Println(lastOutput)
		}
		if err != nil {
			return lastOutput, fmt.Errorf("error in running command: %v", err)
		}
	}
	return lastOutput, nil
}

// uploadToBlobStorage uploads a file to azure storage
//
// NOTE: data limit per block is 100mb - will error if trying to upload something bigger
func uploadToBlobStorage(container *storage.Container, name string, data []byte) error {
	b := container.GetBlobReference(name)
	// if it exists, it will overwrite
	if err := b.CreateBlockBlob(nil); err != nil {
		return err
	}

	// the block just needs some base64 ID that's unique per request, probably
	blockID := base64.StdEncoding.EncodeToString([]byte("0"))
	err := b.PutBlock(blockID, data, nil)
	if err != nil {
		return err
	}

	list, err := b.GetBlockList(storage.BlockListTypeUncommitted, nil)
	if err != nil {
		return err
	}
	uncommittedBlocksList := make([]storage.Block, len(list.UncommittedBlocks))
	for i := range list.UncommittedBlocks {
		uncommittedBlocksList[i].ID = list.UncommittedBlocks[i].Name
		uncommittedBlocksList[i].Status = storage.BlockStatusUncommitted
	}

	// this commits the blocks or something
	err = b.PutBlockList(uncommittedBlocksList, nil)
	if err != nil {
		return err
	}
	return nil
}

func newClusterName(prefix string) string {
	ts := time.Now().UTC().Format("20060102T150405")
	return prefix + "-" + ts
}

func scpFileToInstance(hostname, localPath, remotePath string) error {
	_, err := runCmds([]*exec.Cmd{
		exec.Command("scp",
			"-o", "ConnectTimeout=20",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no",
			localPath,
			fmt.Sprintf("ubuntu@%s:%s", hostname, remotePath)),
	})
	return err

}

func runRemoteCmd(hostname, cmd string) error {
	_, err := runCmds([]*exec.Cmd{
		exec.Command("ssh",
			"-o", "ConnectTimeout=20",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no",
			"ubuntu@"+hostname,
			fmt.Sprintf("bash -c '%s'", cmd)),
	})
	return err
}
