package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

var editors = map[string]bool{
	"atom":      true,
	"intelliji": true,
	"neovim":    true,
	"pycharm":   true,
	"sublime3":  true,
	"vim":       true,
	"vscode":    true,
}

func main() {
	var days int
	var userID string
	flag.StringVar(&userID, "uid", "0", "")
	flag.IntVar(&days, "days", 2, "days of events to retreive")
	flag.Parse()

	listing, err := tracks.List(tracks.Bucket, tracks.ClientEventSource)
	if err != nil {
		log.Fatalln(err)
	}

	var userTracks []*analytics.Track
	for idx, day := range listing.Days {
		if idx < len(listing.Days)-days {
			continue
		}
		r := tracks.NewReader(tracks.Bucket, day.Keys, 8)
		go r.StartAndWait()

		for track := range r.Tracks {
			uid := tracks.ParseUserID(track)
			if uid != userID {
				continue
			}
			userTracks = append(userTracks, track)
		}
	}

	sort.Sort(tracks.ByTimestamp(userTracks))

	for _, track := range userTracks {
		switch track.Event {
		case "Client HTTP Batch":
			fmt.Printf("%s\t%s\t%s\t%s\t%v\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				track.Properties["requests"],
			)
		case "Background Library Walk Completed":
			dirs := int(track.Properties["scanned_dirs"].(float64))
			sinceStart := time.Duration(track.Properties["since_start_ns"].(float64))
			fmt.Printf("%s\t%s\t%s\t%s\t%d\t%s\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				dirs,
				sinceStart,
			)
		case "Index Build":
			artifacts := int(track.Properties["artifacts"].(float64))
			var files int
			if track.Properties["filtered_files"] == nil {
				files = int(track.Properties["files"].(float64))
			} else {
				files = int(track.Properties["filtered_files"].(float64))
			}
			var source string
			if track.Properties["source"] != nil {
				source = track.Properties["source"].(string)
			}
			var waitDuration float64
			if track.Properties["wait_duration_ns"] != nil {
				waitDuration = track.Properties["wait_duration_ns"].(float64)
			}
			sinceStart := time.Duration(track.Properties["since_start_ns"].(float64))
			fmt.Printf("%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				artifacts,
				files,
				source,
				time.Duration(waitDuration),
				sinceStart,
				track.Properties["error"].(string),
			)
		case "Index Build Filtered":
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				track.Properties["reason"],
			)
		case "Local Index Added":
			fmt.Printf("%s\t%s\t%s\t%s\t%v\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				track.Properties["local_code_status"],
			)
		case "Found Editors", "Found Installed Plugins", "Successful Plugin Updates", "Failed Plugin Updates":
			foundEditors := make(map[string][]interface{})
			for e := range editors {
				entry := track.Properties[e]
				if entry != nil {
					foundEditors[e] = entry.([]interface{})
				}
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%v\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				foundEditors,
			)
		case "Editor Plugin Installed":
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				track.Properties["editor"].(string),
				track.Properties["path"].(string),
			)
		case "Editor Plugin Install Failed", "Editor Plugin Uninstall Failed":
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				track.Timestamp,
				track.Event,
				track.Properties["platform"].(string),
				track.Properties["client_version"].(string),
				track.Properties["editor"].(string),
				track.Properties["path"].(string),
				track.Properties["error"].(string),
			)
		default:
			fmt.Printf("%s\t%s\n",
				track.Timestamp,
				track.Event,
			)
		}
	}
}
