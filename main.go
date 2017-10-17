package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var flagSocket = flag.String("socket", "/tmp/rtorrent.sock", "SCGI communication socket.")
var flagOlder = flag.Int("older", 0, "Older than <int> days.")
var flagNewer = flag.Int("newer", 0, "Newer than <int> days.")
var flagSize = flag.Int64("size", 0, "Larger than <int> mb.")
var flagVerbose = flag.Bool("verbose", false, "Verbose output.")
var flagNoLinks = flag.Bool("nolinks", false, "Without hard-links outside base path.")
var flagName = flag.String("name", "", "Regexp which should match the name.")
var flagStop = flag.Bool("stop", false, "Stop matched torrents.")
var flagDelete = flag.Bool("delete", false, "Delete matched torrents.")
var flagSort = flag.String("sort", "date", "Sort order.")

func userConfirm(q string) bool {
	var s string

	fmt.Printf("%s (y/N): ", q)
	if _, err := fmt.Scan(&s); err != nil {
		panic(err)
	}
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" || s == "yes"
}

func main() {
	flag.Parse()
	rt := RTorrent{
		Sock: *flagSocket,
	}

	tor, err := rt.GetTorrents()
	if err != nil {
		log.Printf("%q\n", err)
		return
	}

	pat := regexp.MustCompile(*flagName)
	older := time.Now().Add(time.Duration(int64(*flagOlder) * -24 * time.Hour.Nanoseconds()))
	newer := time.Now().Add(time.Duration(int64(*flagNewer) * -24 * time.Hour.Nanoseconds()))

	var sel []Torrent
	for _, t := range tor {
		if !t.Active {
			continue
		}
		if (*flagOlder != 0 && t.Time.After(older)) || (*flagNewer != 0 && t.Time.Before(newer)) {
			continue
		}
		if t.Size < *flagSize*1024*1024 {
			continue
		}
		if !pat.MatchString(t.Name) {
			continue
		}
		sel = append(sel, t)
	}

	if *flagNoLinks || *flagVerbose {
		// Only load the individual file information if needed.
		rt.GetTorrentFiles(sel)
	}

	if *flagNoLinks {
		var h []Torrent
		for _, t := range sel {
			if t.Links <= uint(len(t.Files)) {
				h = append(h, t)
			}
		}
		sel = h
	}

	if *flagStop {
		for _, t := range sel {
			fmt.Println(t.Path)
		}
		if userConfirm("Confirm stopping these torrents") {
			if err := rt.StopTorrents(sel); err != nil {
				log.Printf("stopping torrents: %q\n", err)
			}
			fmt.Println("Stopping.")
		}

		if *flagDelete && userConfirm("Confirm DELETING these torrents") {
			if err := rt.DeleteTorrents(sel); err != nil {
				log.Printf("deleting torrents: %q\n", err)
			}
			for _, t := range sel {
				fmt.Printf("Deleting %s\n", t.Path)
				os.RemoveAll(t.Path)
			}
		}
		return
	}

	var sf func(int, int) bool
	switch *flagSort {
	case "name":
		sf = func(i, j int) bool { return sel[i].Name < sel[j].Name }
	case "size":
		sf = func(i, j int) bool { return sel[i].Size < sel[j].Size }
	default:
		sf = func(i, j int) bool { return sel[i].Time.Unix() < sel[j].Time.Unix() }
	}
	sort.Slice(sel, sf)

	tot := int64(0)
	for _, t := range sel {
		fst := "2006.01.02"
		if *flagVerbose {
			fst = "2006.01.02-15:04:05"
		}
		fmt.Printf("%s %6dMB %s\n", t.Time.Format(fst), t.Size/1024/1024, t.Name)
		if *flagVerbose {
			fmt.Printf("\tHash: %s\n\tFiles: %d\n", t.Hash, len(t.Files))
		}
		tot += t.Size
	}
	fmt.Printf("Total: %d torrents, %.02fGB\n", len(sel), float32(tot)/1024/1024/1024)

	//     for _, i := range sel {
	//         fmt.Printf("%q\n", i)
	//     }
}
