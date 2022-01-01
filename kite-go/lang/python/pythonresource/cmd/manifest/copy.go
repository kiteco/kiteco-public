package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// TODO(naman) we could probably also use a workerpool here for potential additional speedups
type copyManager struct {
	up   *s3manager.Uploader
	down *s3manager.Downloader
	cli  *s3.S3
}

func newCopyManager() *copyManager {
	return &copyManager{}
}

func (c *copyManager) client() (*s3.S3, error) {
	if c.cli == nil {
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		c.cli = s3.New(sess, aws.NewConfig().WithRegion("us-west-1")) // TODO(naman) avoid hardcoding region?
	}
	return c.cli, nil
}
func (c *copyManager) uploader() (*s3manager.Uploader, error) {
	if c.up == nil {
		cli, err := c.client()
		if err != nil {
			return nil, err
		}
		c.up = s3manager.NewUploaderWithClient(cli)
	}
	return c.up, nil
}
func (c *copyManager) downloader() (*s3manager.Downloader, error) {
	if c.down == nil {
		cli, err := c.client()
		if err != nil {
			return nil, err
		}
		c.down = s3manager.NewDownloaderWithClient(cli)
	}
	return c.down, nil
}

func (c *copyManager) slowCopy(src, dst string) error {
	srcF, err := fileutil.NewCachedReader(string(src))
	if err != nil {
		return err
	}
	defer srcF.Close()

	dstF, err := fileutil.NewBufferedWriter(string(dst))
	if err != nil {
		return err
	}
	defer dstF.Close()

	_, err = io.Copy(dstF, srcF)
	return err
}
func (c *copyManager) fastLocalCopy(src, dst string) error {
	return os.Link(src, dst)
}
func (c *copyManager) localCopy(src, dst string) error {
	// sanity checks
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	dfi, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && os.SameFile(sfi, dfi) {
		return nil
	}

	if err = os.Link(src, dst); err == nil {
		return nil
	}
	return c.slowCopy(src, dst)
}

func (c *copyManager) s3copy(src, dst *url.URL) error {
	cli, err := c.client()
	if err != nil {
		return err
	}
	_, err = cli.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(dst.Host),
		Key:        aws.String(strings.TrimPrefix(dst.Path, "/")),
		CopySource: aws.String(fmt.Sprintf("/%s/%s", src.Host, strings.TrimPrefix(src.Path, "/"))),
	})
	return err
}
func (c *copyManager) s3upload(src string, dst *url.URL) error {
	up, err := c.uploader()
	if err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	_, err = up.Upload(&s3manager.UploadInput{
		Bucket: aws.String(dst.Host),
		Key:    aws.String(strings.TrimPrefix(dst.Path, "/")),
		Body:   f,
	})
	return err
}
func (c *copyManager) s3download(src *url.URL, dst string) error {
	down, err := c.downloader()
	if err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = down.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(src.Host),
		Key:    aws.String(strings.TrimPrefix(src.Path, "/")),
	})
	return err
}
func (c *copyManager) copy(src, dst string) error {
	s3src, _ := awsutil.ValidateURI(src)
	s3dst, _ := awsutil.ValidateURI(dst)

	switch {
	case s3src != nil && s3dst != nil:
		return c.s3copy(s3src, s3dst)
	case s3src != nil:
		return c.s3download(s3src, dst)
	case s3dst != nil:
		return c.s3upload(src, s3dst)
	default:
		return c.localCopy(src, dst)
	}
}
