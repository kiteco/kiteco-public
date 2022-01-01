package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var outpath string
	flag.StringVar(&outpath, "output", "", "path to which to write output")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if outpath == "" {
		log.Fatal("Must specify output path by --output")
	}

	rankingDB := curation.GormDB(
		envutil.MustGetenv("RANKING_DB_DRIVER"),
		envutil.MustGetenv("RANKING_DB_URI"))
	rankingManager := ranking.NewQueryManager(rankingDB)

	// Get all rankings
	labels, err := rankingManager.GetAllLabels()
	if err != nil {
		log.Fatal(err)
	}
	seen := make(map[int64]struct{})

	for i, label := range labels {
		if i%1000 == 0 {
			log.Printf("Processed %d out of %d ranking labels\n", i, len(labels))
		}
		if _, exists := seen[label.SnapshotID]; !exists {
			seen[label.SnapshotID] = struct{}{}
		}
	}

	// Open the output stream
	f, err := os.Create(outpath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	for id := range seen {
		f.WriteString(strconv.FormatInt(id, 10) + "\n")
	}

	log.Printf("Wrote %d snapshot ids\n", len(seen))
}
