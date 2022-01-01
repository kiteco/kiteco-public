package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const (
	remoteBundleFile = "/var/kite/upload/bundle.tar.gz"
	deployScriptName = "deploy-bundle.sh"
	sshKeyToUpload   = "kite-dev-azure"
)

func deployBundle(bundleFile string, clusterName string, cleanupClusters []string) error {
	tmpDir, err := ioutil.TempDir("", "azure-cluster-deploy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	localDeployPath := path.Join(tmpDir, deployScriptName)
	if err := writeDeployFile(localDeployPath, cleanupClusters); err != nil {
		return fmt.Errorf("error writing %s: %v", localDeployPath, err)
	}

	remoteDeployPath := path.Join("/var/kite/upload", deployScriptName)

	ips, err := getClusterIPs(clusterName)
	if err != nil {
		return fmt.Errorf("error getting cluster IPs: %v", err)
	}

	homePath := os.Getenv("HOME")
	if homePath == "" {
		return fmt.Errorf("HOME env variable not set")
	}

	sshKey := path.Join(homePath, ".ssh", sshKeyToUpload)

	log.Printf("deploying %s to %d instances in cluster %s", bundleFile, len(ips), clusterName)

	var m sync.Mutex
	var completed int32

	wp := workerpool.New(1)
	for idx, ip := range ips {
		localIdx := idx
		localIP := ip
		wp.Add([]workerpool.Job{func() error {
			// give each instance in the cluster a unique ID within that cluster
			identifyCmd := fmt.Sprintf("echo %d > /var/kite/instance-id && echo %d > /var/kite/instance-count", localIdx, len(ips))
			if err := runRemoteCmd(localIP, identifyCmd); err != nil {
				return fmt.Errorf("%s: error running %s: %v", localIP, identifyCmd, err)
			}

			if err := scpFileToInstance(localIP, sshKey, path.Join("/home/ubuntu/.ssh", sshKeyToUpload)); err != nil {
				return err
			}

			if err := scpFileToInstance(localIP, bundleFile, remoteBundleFile); err != nil {
				return err
			}

			if err := scpFileToInstance(localIP, localDeployPath, remoteDeployPath); err != nil {
				return err
			}

			runBundleCmd := fmt.Sprintf("nohup bash %s > /var/kite/log/%s.log 2>&1 &", remoteDeployPath, deployScriptName)
			if err := runRemoteCmd(localIP, runBundleCmd); err != nil {
				return fmt.Errorf("%s: error running %s: %v", localIP, runBundleCmd, err)
			}

			m.Lock()
			defer m.Unlock()
			completed++
			log.Printf("(%d/%d) %s \xE2\x9C\x94", completed, len(ips), localIP)

			return nil
		}})
	}

	err = wp.Wait()
	if err != nil {
		return err
	}

	wp.Stop()

	log.Printf("deploy to cluster %s completed", clusterName)

	return nil
}

func writeDeployFile(deployPath string, cleanupClusters []string) error {
	outf, err := os.Create(deployPath)
	if err != nil {
		return err
	}
	defer outf.Close()

	return templates.RenderText(outf, "deploy-bundle.sh", map[string]interface{}{
		"CleanupClusters": cleanupClusters,
	})
}
