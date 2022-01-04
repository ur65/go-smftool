package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ur65/go-smftool"
)

func baseWithoutExt(path string) string {
	base := filepath.Base(path)
	return base[:len(base)-len(filepath.Ext(path))]
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [OPTION] SMF_FILE [TRACK_NUM1 TRACK_NUM2]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Swap track TRACK_NUM1 and track TRACK_NUM2 in SMF_FILE.\n\n")
		flag.PrintDefaults()
		os.Exit(0)
	}
}

var (
	hflag = flag.Bool("h", false, "show this message")
	lflag = flag.Bool("l", false, "show track list")
	oflag = flag.String("o", "", "dst file name")
)

func run() error {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 || *hflag {
		flag.Usage()
	}

	fpath := args[0]
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	if *lflag {
		s, err := smftool.Decode(f)
		if err != nil {
			return err
		}
		for i, track := range s.Tracks[1:] {
			fmt.Printf("[%d] %s\n", i+1, track.Events[1].Value)
		}
		return nil
	}

	if len(args) < 3 {
		flag.Usage()
	}

	a, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	b, err := strconv.Atoi(args[2])
	if err != nil {
		return err
	}

	outpath := *oflag
	if outpath == "" {
		outpath = fmt.Sprintf("%s_swap_%d_%d%s", baseWithoutExt(fpath), a, b, filepath.Ext(fpath))
	}

	dst, err := os.Create(outpath)
	if err != nil {
		return err
	}

	f.Seek(0, io.SeekStart)
	if err := smftool.SwapTrack(dst, f, a, b); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
