package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"dedup/app"
	"dedup/fs"
	"dedup/fs/mockfs"
	"dedup/fs/realfs"
)

func main() {
	logName := os.Getenv("DEDUP_LOG")
	if logName != "" {
		logFile, err := os.Create(logName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer logFile.Close()

		log.SetOutput(logFile)
		log.SetFlags(log.Lmicroseconds)
	} else {
		log.SetOutput(io.Discard)
	}

	var fsys fs.FS
	if len(os.Args) != 2 {
		fmt.Println("Provide path to an archive")
		os.Exit(1)
	}
	if os.Args[1] == "-sim" {
		fsys = mockfs.New("origin")
	} else {
		path, err := realfs.AbsPath(os.Args[1])
		if err != nil {
			log.Printf("Failed to scan archives: %W\n", err)
			panic(err)
		}
		fsys = realfs.New(path)
	}

	app.Run(fsys)
}
