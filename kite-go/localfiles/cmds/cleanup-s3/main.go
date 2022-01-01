package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const contentHashSet = "s3://kite-data/localfiles/2017-06-05/contenthashes.gob.gz"
const filterTs = "2018-04-11"

var (
	dumpfiles = map[string]string{
		"us-west-1": "hashes-uswest1.csv",
		"us-east-1": "hashes-useast1.csv",
		"eu-west-1": "hashes-euwest1.csv",
	}
	buckets = map[string]string{
		"us-west-1": "kite-local-content",
		"us-east-1": "kite-local-content-us-east-1",
		"eu-west-1": "kite-local-content-eu-west-1",
	}
)

func main() {
	var region string
	var del bool
	flag.StringVar(&region, "region", "us-west-1", "")
	flag.BoolVar(&del, "delete", false, "")
	flag.Parse()

	ts, err := time.Parse("2006-01-02", filterTs)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Filter ts: %s\n", ts)

	hashset, err := loadContentHashSet(contentHashSet)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Loaded %d hashes from content hash set\n", len(hashset))

	files, err := loadDatabaseDump(dumpfiles[region])
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Loaded %d hashes from %s dump\n", len(files), region)

	var inDb, inHashSet, tooRecent int
	toDelete := make(map[string]bool)
	lastModifiedMonth := make(map[string]int)
	bucket := buckets[region]
	err = scan(region, bucket, toDelete, func(obj *s3.Object, page int) {
		if obj.LastModified == nil {
			return
		}
		m := obj.LastModified.Format("2006-01-02")
		lastModifiedMonth[m]++
		h := *(obj.Key)
		if files[h] {
			inDb++
		}
		if hashset[h] {
			inHashSet++
		}
		if !hashset[h] && !files[h] {
			if obj.LastModified.Before(ts) {
				toDelete[h] = true
				if !del {
					log.Println(h, obj.LastModified)
				}
			} else {
				tooRecent++
			}
		}
		if page%100 == 0 {
			dumpMonths(lastModifiedMonth)
		}
	})
	if err != nil {
		log.Println(err)
	}
	log.Printf("%d in db, %d in hash set, %d too recent", inDb, inHashSet, tooRecent)
	log.Printf("Going to delete %d of %d hashes\n", len(toDelete), len(files))
	if del {
		deleteHashes(region, bucket, toDelete)
	}
}

func dumpMonths(months map[string]int) {
	type monthCount struct {
		month string
		count int
	}

	var total int
	var counts []monthCount
	for m, c := range months {
		total += c
		counts = append(counts, monthCount{m, c})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].month < counts[j].month
	})

	f, err := os.Create("months")
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	for _, c := range counts {
		fmt.Fprintf(f, "%s %.02f %d\n", c.month, float64(c.count)/float64(total), c.count)
	}
}

func scan(region, bucket string, toDelete map[string]bool, fn func(obj *s3.Object, page int)) error {
	svc, err := awsutil.NewS3(region)
	if err != nil {
		log.Fatalln(err)
	}

	listReq := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	var pages, total int
	err = svc.ListObjectsV2Pages(listReq, func(listResp *s3.ListObjectsV2Output, last bool) bool {
		pages++
		total += len(listResp.Contents)
		if pages%100 == 0 {
			fmt.Printf("Page %d, to delete %d of %d hashes so far\n", pages, len(toDelete), total)
		}
		for _, obj := range listResp.Contents {
			fn(obj, pages)
		}
		return true
	})
	fmt.Printf("Page %d, to delete %d of %d hashes so far\n", pages, len(toDelete), total)
	return err
}

const objsPerDelete = 1000

func deleteHashes(region, bucket string, todelete map[string]bool) error {

	svc, err := awsutil.NewS3(region)
	if err != nil {
		log.Fatalln(err)
	}

	total := len(todelete)
	var deleted int
	for len(todelete) > 0 {
		deleteObjsReq := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{},
		}

		var objects []*s3.ObjectIdentifier
		for key := range todelete {
			objects = append(objects, &s3.ObjectIdentifier{
				Key: aws.String(key),
			})
			if len(objects) >= objsPerDelete {
				break
			}
		}

		deleteObjsReq.Delete.SetObjects(objects)
		_, err = svc.DeleteObjects(deleteObjsReq)
		if err != nil {
			return err
		}

		deleted += len(objects)
		for _, obj := range objects {
			delete(todelete, *obj.Key)
		}

		log.Printf("deleted %d objects (%.02f)", deleted, float64(deleted)/float64(total))
	}
	return nil
}

func loadContentHashSet(path string) (map[string]bool, error) {
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gunzip, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	hashes := make(map[string]bool)
	err = gob.NewDecoder(gunzip).Decode(&hashes)
	if err != nil {
		return nil, err
	}

	return hashes, nil
}

func loadDatabaseDump(path string) (map[string]bool, error) {
	hashes := make(map[string]bool)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) > 0 {
			hashes[record[0]] = true
		}
	}
	return hashes, nil
}
