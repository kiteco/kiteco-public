package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	chart "github.com/wcharczuk/go-chart"
)

func main() {
	var days int
	var old bool
	flag.IntVar(&days, "days", 3, "days of events to receive")
	flag.BoolVar(&old, "old", false, "use old Client Event source")
	flag.Parse()

	source := tracks.ClientEventSource
	if old {
		source = tracks.OldClientEventSource
	}
	listing, err := tracks.List(tracks.Bucket, source)
	if err != nil {
		log.Fatalln(err)
	}

	for idx, day := range listing.Days {
		if idx < len(listing.Days)-days {
			continue
		}

		r := tracks.NewReader(tracks.Bucket, day.Keys, 8)
		go r.StartAndWait()
		var total, local int
		for track := range r.Tracks {
			if track.Event != "Client HTTP Batch" {
				continue
			}

			total++
			kl := track.Properties["kite_local"]
			if kl == nil {
				continue
			}
			if kl.(bool) {
				local++
			}
		}

		percent := (float64(local) / float64(total)) * 100
		log.Println("building report for", day.Day.Format("2006-01-02"))
		log.Printf("Total: %d, Kite Local: %d, %%: %f", total, local, percent)
		log.Println("---")

		daysList = append(daysList, day.Day)
		percents = append(percents, percent)
	}
	graphPercentLocal()
}

var (
	daysList []time.Time
	percents []float64
)

func graphPercentLocal() {
	series := []chart.Series{
		chart.TimeSeries{
			Name:    "percent local",
			XValues: daysList,
			YValues: percents,
		},
	}

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Name:      "count",
			NameStyle: chart.StyleShow(),
			Style: chart.Style{
				Show: true,
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
		},
		Series: series,
	}

	f, err := os.Create("percent-events-kite-local.png")
	if err != nil {
		log.Fatalln(err)
	}

	graph.Render(chart.PNG, f)

	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
