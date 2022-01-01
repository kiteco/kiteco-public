package awsutil

import (
	"bytes"
	"fmt"
	"io"
	"net/url"

	"github.com/kiteco/kiteco/kite-golib/azureutil"
)

func mirroredAzureReader(s3url *url.URL, s3Checksum []byte) (io.ReadCloser, error) {
	container, path := azureContainerPath(s3url)

	azureChecksum, err := azureutil.GetBlobChecksum(container, path)
	if err != nil {
		return nil, fmt.Errorf("error finding Azure checksum for container %s, path %s: %v",
			container, path, err)
	}

	if bytes.Compare(s3Checksum, azureChecksum) != 0 {
		return nil, fmt.Errorf("s3 (%s) and azure (%s) checksums disagree for uri %s",
			string(s3Checksum), string(azureChecksum), s3url.String())
	}

	r, err := azureutil.NewBlobReader(container, path)
	if err != nil {
		return nil, fmt.Errorf("error getting blob reader: %v", err)
	}
	return r, nil
}

func copyToAzure(s3url *url.URL, isGZipped bool) error {
	r, err := objectReader(s3url, true)
	if err != nil {
		return err
	}
	defer r.Close()

	container, path := azureContainerPath(s3url)

	w, err := azureutil.NewBlobWriter(container, path, isGZipped)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	if err != nil {
		w.Close()
		return err
	}

	return w.Close()
}

func azureContainerPath(s3uri *url.URL) (string, string) {
	return s3uri.Host, s3uri.Path[1:]
}
