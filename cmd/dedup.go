package main

import (
	"fmt"
	"log"
	"os"

	"dedup/app"
	"dedup/fs"
	"dedup/fs/mockfs"
	"dedup/fs/realfs"
	"dedup/lifecycle"
)

func main() {
	logFile, err := os.Create("dedup.log")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.Lmicroseconds)

	var lc = lifecycle.New()

	var fsys fs.FS
	if len(os.Args) != 2 {
		fmt.Println("Provide path to an archive")
		os.Exit(1)
	}
	if os.Args[1] == "-sim" {
		fsys = mockfs.New("origin", lc)
	} else {
		path, err := realfs.AbsPath(os.Args[1])
		if err != nil {
			log.Printf("Failed to scan archives: %W\n", err)
			panic(err)
		}
		fsys = realfs.New(path, lc)
	}

	app.Run(fsys)
}
