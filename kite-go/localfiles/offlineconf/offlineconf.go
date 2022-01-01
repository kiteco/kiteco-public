package offlineconf

import (
	"compress/gzip"
	"io/ioutil"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jmoiron/sqlx"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/diskcache"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq" // Importing this allows sqlx to use the postgres driver.
)

//- FileManager

var (
	dbDriver = envutil.GetenvDefault("LOCALFILES_DB_DRIVER", "postgres")

	uriByRegion = map[string]string{
		// AWS
		"us-west-1": envutil.GetenvDefault("LOCALFILES_DB_URI_US_WEST_1", ""),
		"us-west-2": envutil.GetenvDefault("LOCALFILES_DB_URI_US_WEST_2", ""),
		// "us-east-1":      envutil.GetenvDefault("LOCALFILES_DB_URI_US_EAST_1", ""),
		// "eu-west-1":      envutil.GetenvDefault("LOCALFILES_DB_URI_EU_WEST_1", ""),
		// "ap-southeast-1": envutil.GetenvDefault("LOCALFILES_DB_URI_AP_SOUTHEAST_1", ""),
		// Azure
		// "eastus":           envutil.GetenvDefault("LOCALFILES_DB_URI_EASTUS", ""),
		"westus2": envutil.GetenvDefault("LOCALFILES_DB_URI_WESTUS2", ""),
		// "azure-westeurope": envutil.GetenvDefault("LOCALFILES_DB_URI_AZURE_WESTEUROPE", ""),
	}

	// TODO these only work for us-west-* and westus* regions
	s3Bucket  = envutil.GetenvDefault("LOCALFILES_S3_BUCKET", "kite-local-content")
	awsRegion = envutil.GetenvDefault("AWS_REGION", "us-west-1")

	dbMap sync.Map
)

// GetFileDB returns a sqlx.DB connection to the files database for the given region, based on environment variables.
func GetFileDB(region string) *sqlx.DB {
	if v, ok := dbMap.Load(region); ok {
		return v.(*sqlx.DB)
	}

	if uri := uriByRegion[region]; uri != "" {
		log.Printf("localfiles/offlineconf: connecting to files database for region: %s", region)
		fdb := localfiles.FileDB(dbDriver, uri)
		dbMap.Store(region, fdb)
		return fdb
	}

	return nil
}

// GetFileManager returns a FileManager for the provided region.
func GetFileManager(region string) *localfiles.FileManager {
	if fdb := GetFileDB(region); fdb != nil {
		return localfiles.NewFileManager(fdb)
	}

	return nil
}

//- FileGetter

type s3Getter struct {
	region     string
	bucketName string
}

// Get implements localcode.FileGetter
func (g s3Getter) Get(key string) ([]byte, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	cli := s3.New(sess, aws.NewConfig().WithRegion(g.region))

	input := &s3.GetObjectInput{
		Bucket: aws.String(g.bucketName),
		Key:    aws.String(key),
	}
	output, err := cli.GetObject(input)
	if err != nil {
		return nil, err
	}
	decomp, err := gzip.NewReader(output.Body)
	if err != nil {
		return nil, err
	}
	defer decomp.Close()
	return ioutil.ReadAll(decomp)
}

// GetFileGetter returns a FileGetter for the provided region, with the specified caching options
func GetFileGetter(region string, cacheOpts *diskcache.Options) (localcode.FileGetter, error) {
	g := localcode.FileGetter(s3Getter{
		region:     awsRegion,
		bucketName: s3Bucket,
	})
	if cacheOpts != nil {
		cache, err := diskcache.OpenTemp(*cacheOpts)
		if err != nil {
			return nil, err
		}
		g = localcode.NewCachedFileGetter(cache, g)
	}
	return g, nil
}
