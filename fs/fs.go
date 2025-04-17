package fs

import "time"

type Events interface {
	Send(msg any)
}

type FS interface {
	Root() string
	Scan(events Events)
	Remove(path string, events Events)
}

type FileMeta struct {
	Idx     int
	Path    string
	Size    int
	ModTime time.Time
	Hash    string
}

// Events

type FileMetas []FileMeta

type FileHashed struct {
	Path string
	Hash string
}

type ArchiveHashed struct {
}
