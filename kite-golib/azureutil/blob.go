package azureutil

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

const (
	// maxBlockSize allowed by the Azure blob storage API
	maxBlockSize = 100 * 1024 * 1024 // 100 MiB
	// the Azure blob storage account name
	storageNameEnvVar = "KITE_AZUREUTIL_STORAGE_NAME"
	// the key for the corresponding blob storage account
	storageKeyEnvVar = "KITE_AZUREUTIL_STORAGE_KEY"
)

var (
	storageName = envutil.GetenvDefault(storageNameEnvVar, "")
	storageKey  = envutil.GetenvDefault(storageKeyEnvVar, "")
)

// GetBlobChecksum returns the MD5 checksum of a blob in hex. Given the same file contents, this should be the same
// checksum as is calculated by md5.New() and by S3.
func GetBlobChecksum(container, path string) ([]byte, error) {
	bs, err := getBlobService()
	if err != nil {
		return nil, err
	}

	ctn := bs.GetContainerReference(container)
	blob := ctn.GetBlobReference(path)

	if err := blob.GetProperties(nil); err != nil {
		return nil, err
	}

	checksum, err := base64.StdEncoding.DecodeString(blob.Properties.ContentMD5)
	if err != nil {
		return nil, err
	}

	hexChecksum := make([]byte, hex.EncodedLen(len(checksum)))
	hex.Encode(hexChecksum, checksum)
	return hexChecksum, nil
}

// NewBlobWriter writes to a with the specified container/path
// gzipped should be true if the contents that will be written are gzipped
func NewBlobWriter(container, path string, gzipped bool) (io.WriteCloser, error) {
	f, err := ioutil.TempFile("", "blobbuffer")
	if err != nil {
		return nil, err
	}
	h := md5.New()
	w := io.MultiWriter(f, h)

	return bufferedBlobWriter{
		container: container,
		path:      path,
		f:         f,
		h:         h,
		w:         w,
		gzipped:   gzipped,
	}, nil
}

type bufferedBlobWriter struct {
	container string
	path      string

	f *os.File
	h hash.Hash
	w io.Writer

	gzipped bool
}

// Write implements io.WriteCloser
func (w bufferedBlobWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

// Close implements io.WriteCloser
func (w bufferedBlobWriter) Close() error {
	defer os.Remove(w.f.Name()) // delete the buffer file from disk
	defer w.f.Close()           // after closing the buffer file handle

	if err := w.f.Sync(); err != nil { // flush to disk
		return err
	}

	_, err := w.f.Seek(0, 0) // seek to beginning
	if err != nil {
		return err
	}

	bs, err := getBlobService()
	if err != nil {
		return err
	}

	// get a reference to the container, creating it if it doesn't exist
	ctn := bs.GetContainerReference(w.container)
	if _, err := ctn.CreateIfNotExists(nil); err != nil {
		return err
	}

	blob := ctn.GetBlobReference(w.path)

	// create an empty blob
	if err := blob.CreateBlockBlob(nil); err != nil {
		return err
	}

	// get the blob etag, and make sure it still refers to an empty blob
	if err := blob.GetProperties(nil); err != nil {
		return err
	}
	etag := blob.Properties.Etag
	if blob.Properties.ContentLength != 0 {
		return fmt.Errorf("blob has non-empty length (%d) before writing", blob.Properties.ContentLength)
	}

	md5Hash := base64.StdEncoding.EncodeToString(w.h.Sum(nil))

	// upload the blocks
	var blocks []storage.Block
	buf := make([]byte, maxBlockSize)
	for true {
		idx := len(blocks)
		// the block IDs must be in increasing alpha order, hence the leading zeros
		// Also prepend with the MD5 hash since the (MD5, block ID) uniquely identify the contents
		blockID := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s-%07d", md5Hash, idx)))

		n, err := io.ReadFull(w.f, buf)

		if err == io.EOF {
			// no bytes were read, we can skip this block
			break
		} else if err != nil && err != io.ErrUnexpectedEOF {
			return err
		}

		if err := blob.PutBlock(blockID, buf[:n], nil); err != nil {
			return err
		}

		blocks = append(blocks, storage.Block{
			ID:     blockID,
			Status: storage.BlockStatusUncommitted,
		})
	}

	// set the MD5 hash explicitly because the way in which Azure calculates the hash by default differs from the way
	// we do and the way S3 does, and so we can't compare hashes otherwise
	blob.Properties.ContentMD5 = md5Hash
	if w.gzipped {
		// also, if the content is gzipped, set the appropriate header so that the file is decompressed automatically
		// upon reading
		blob.Properties.ContentEncoding = "gzip"
	}

	// commit the blocks to the file; fail if the etag no longer matches
	if err := blob.PutBlockList(blocks, &storage.PutBlockListOptions{IfMatch: etag}); err != nil {
		return err
	}

	return nil
}

func getBlobService() (storage.BlobStorageClient, error) {
	if storageName == "" {
		return storage.BlobStorageClient{}, fmt.Errorf("%s env var needs to be set", storageNameEnvVar)
	}
	if storageKey == "" {
		return storage.BlobStorageClient{}, fmt.Errorf("%s env var needs to be set", storageKeyEnvVar)
	}

	sc, err := storage.NewBasicClient(storageName, storageKey)
	if err != nil {
		return storage.BlobStorageClient{}, err
	}
	return sc.GetBlobService(), nil
}
