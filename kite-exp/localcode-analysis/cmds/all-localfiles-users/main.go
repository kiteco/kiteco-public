package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq"
)

const batchSize = 100

func main() {
	args := struct {
		Out string `arg:"required"`
	}{}
	arg.MustParse(&args)

	dbUri := envutil.MustGetenv("PROD_WESTUS2_LOCALFILES_DB_URI")
	db := localfiles.FileDB("postgres", dbUri)

	var maxUserID int64
	query := "SELECT MAX(user_id) FROM file"
	if err := db.Get(&maxUserID, query); err != nil {
		log.Fatal(err)
	}
	log.Printf("max user ID: %d", maxUserID)

	outf, err := os.Create(args.Out)
	if err != nil {
		log.Fatal(err)
	}
	defer outf.Close()

	var totalResults int

	for uid := int64(0); uid <= maxUserID; uid += batchSize {
		type result struct {
			UserID  int64  `db:"user_id"`
			Machine string `db:"machine"`
			Count   int64  `db:"count"`
		}

		query := "SELECT user_id, machine, COUNT(*) AS count FROM file WHERE user_id >= $1 AND user_id < $2" +
			" GROUP BY user_id, machine"
		var results []result
		if err := db.Select(&results, query, uid, uid+batchSize); err != nil {
			log.Fatal(err)
		}

		var resultFiles int64
		for _, res := range results {
			resultFiles += res.Count
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].UserID == results[j].UserID {
				return results[i].Machine < results[j].Machine
			}
			return results[i].UserID < results[j].UserID
		})

		for _, r := range results {
			if _, err := fmt.Fprintf(outf, "%d,%s,%d\n", r.UserID, r.Machine, r.Count); err != nil {
				log.Fatal(err)
			}
		}

		totalResults += len(results)

		log.Printf("user id = %d/%d, results = %d, files in results = %d, total users/machines = %d",
			uid, maxUserID, len(results), resultFiles, totalResults)
	}

	log.Printf("wrote %d users/machines to %s", totalResults, args.Out)
}
