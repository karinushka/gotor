package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/kolo/xmlrpc"
	scgi "github.com/mpl/scgiclient"
)

type RTorrent struct {
	Sock string
}

type Torrent struct {
	Name   string
	Path   string
	Hash   string
	Date   time.Time
	Time   time.Time
	Size   int64
	Active bool
	Files  []string
	Links  uint
}

type Method struct {
	MethodName string
	Params     []interface{}
}

func (p *RTorrent) Call(method string, args []interface{}) ([][]interface{}, error) {

	req, err := xmlrpc.EncodeMethodCall(method, args...)
	if err != nil {
		return nil, fmt.Errorf("Encoding: %q\n", err)
	}
	// Need to replace these string because Go considers lowercase to be
	// private fields in structures and will not marshall them into XML.
	if strings.Index(method, "multicall") > -1 {
		req = bytes.Replace(req, []byte("MethodName"), []byte("methodName"), -1)
		req = bytes.Replace(req, []byte("Params"), []byte("params"), -1)
	}

	// log.Printf("multicall: %s\n", req)
	rep, err := scgi.Send(p.Sock, bytes.NewReader(req))
	if err != nil {
		return nil, fmt.Errorf("Sending: %q\n", err)
	}

	// log.Printf("body: %q\n", rep.Body)

	resp := xmlrpc.NewResponse(rep.Body)
	var result [][]interface{}
	if err := resp.Unmarshal(&result); err != nil {
		return nil, fmt.Errorf("Unmarshalling: %q\n", err)
	}

	return result, nil
}

func (p *RTorrent) MultiCall(methods []Method) ([][]interface{}, error) {
	//     methods := []Method{
	//         Method{MethodName: "d.name", Params: []interface{}{"63E9359CA3542A335EC64EAF77822A1326D4D8DB"}},
	//         Method{MethodName: "f.path", Params: []interface{}{"C6B21C75287BF2B948FED8FC5B5F613251DA10AB", 0}},
	//         Method{MethodName: "f.multicall", Params: []interface{}{"63E9359CA3542A335EC64EAF77822A1326D4D8DB", 0, "f.path="}},
	//     }
	return p.Call("system.multicall", []interface{}{methods})
}

func (p *RTorrent) GetTorrents() ([]Torrent, error) {

	args := []interface{}{
		"", // None-existing hash placeholder
		"main",
		"d.name=",
		"d.hash=",
		"d.creation_date=",
		"d.size_bytes=",
		"d.is_active=",
		"d.base_path=",
	}
	res, err := p.Call("d.multicall2", args)
	if err != nil {
		return nil, err
	}

	var tor []Torrent
	for _, r := range res {
		//         fmt.Printf("%q\n", r)
		t := Torrent{
			Name:   r[0].(string),
			Hash:   r[1].(string),
			Date:   time.Unix(r[2].(int64), 0),
			Size:   r[3].(int64),
			Active: (1 == r[4].(int64)),
		}
		if t.Active {
			t.Path = r[5].(string)
			if fi, err := os.Stat(t.Path); err == nil {
				t.Time = fi.ModTime()
			}
		}
		tor = append(tor, t)
	}
	return tor, nil
}

// Returns amount of hardlinks present in this torrent.
func loadHardlinks(t *Torrent) {
	bi, err := os.Lstat(t.Path)
	if err != nil {
		log.Printf("reading %q: %q\n", t.Path, err)
		return
	}

	var files []string
	if bi.IsDir() {
		for _, f := range t.Files {
			files = append(files, path.Join(t.Path, f))
		}
	} else {
		files = append(files, t.Path)
	}

	for _, f := range files {
		fi, err := os.Lstat(f)
		if err != nil {
			log.Printf("reading %q link count: %q\n", f, err)
		}
		// https://github.com/docker/docker/blob/master/pkg/archive/archive_unix.go
		// in 'func setHeaderForSpecialDevice()'
		s, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			log.Printf("cannot convert stat value to syscall.Stat_t")
		}
		// Total number of files/hardlinks connected to this file's inode:
		t.Links += uint(s.Nlink)
	}
}

func (p *RTorrent) GetTorrentFiles(tors []Torrent) error {
	var methods []Method
	for _, t := range tors {
		methods = append(methods, Method{
			MethodName: "f.multicall", Params: []interface{}{t.Hash, 0, "f.path="},
		})
	}
	res, err := p.MultiCall(methods)
	if err != nil {
		return err
	} else {
		for i, r := range res {
			var files []string
			for _, s := range r[0].([]interface{}) {
				f := s.([]interface{})[0].(string)
				files = append(files, f)
			}
			tors[i].Files = files
			loadHardlinks(&tors[i])
			//             fmt.Printf("%s: %q\n", tors[i].Name, files)
		}
	}
	return nil
}

func (p *RTorrent) StopTorrents(tors []Torrent) error {
	var methods []Method
	for _, t := range tors {
		methods = append(methods, Method{
			MethodName: "d.stop", Params: []interface{}{t.Hash},
		})
	}
	_, err := p.MultiCall(methods)
	return err
}

func (p *RTorrent) DeleteTorrents(tors []Torrent) error {
	var methods []Method
	for _, t := range tors {
		methods = append(methods, Method{
			MethodName: "d.erase", Params: []interface{}{t.Hash},
		})
	}
	_, err := p.MultiCall(methods)
	return err
}
