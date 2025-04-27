package realfs

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	iofs "io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/text/unicode/norm"

	"dedup/fs"
)

const hashFileName = ".meta.csv"
const bufSize = 256 * 1024

type meta struct {
	inode uint64
	file  *fs.FileMeta
}

type FS struct {
	root string
}

func New(path string) *FS {
	return &FS{root: path}
}

func (fsys *FS) Root() string {
	return fsys.root
}

func (fsys *FS) Remove(path string) {
	err := os.Remove(filepath.Join(fsys.root, path))
	if err != nil {
		log.Printf("failed to remove file %q: %#v", path, err)
	}
	log.Println("removed", path)
}

func (fsys *FS) scan(events fs.Events) {
	metas := fs.FileMetas{}

	metaMap := fsys.readMeta()
	var metaSlice []*meta

	defer func() {
		_ = fsys.storeMeta(fsys.root, metaSlice)
		events.Send(fs.ArchiveHashed{})
	}()

	osfs := os.DirFS(fsys.root)
	err := iofs.WalkDir(osfs, ".", func(path string, d iofs.DirEntry, err error) error {
		if d.IsDir() && strings.HasPrefix(d.Name(), "~~~") {
			return iofs.SkipDir
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		if err != nil {
			log.Printf("Error: failed to scan archive %q: %#v\n", fsys.root, err)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			log.Printf("Error: failed to scan archive %q: %#v\n", fsys.root, err)
			return nil
		}

		size := int(info.Size())
		if size == 0 {
			return nil
		}

		modTime := info.ModTime()
		modTime = modTime.UTC().Round(time.Second)

		file := &fs.FileMeta{
			Path:    norm.NFC.String(path),
			Size:    size,
			ModTime: modTime,
		}

		sys := info.Sys().(*syscall.Stat_t)
		readMeta := metaMap[sys.Ino]
		if readMeta != nil && readMeta.ModTime == modTime && readMeta.Size == size {
			file.Hash = readMeta.Hash
		}

		metas = append(metas, *file)

		metaSlice = append(metaSlice, &meta{
			inode: sys.Ino,
			file:  file,
		})
		metaMap[sys.Ino] = file

		return nil
	})

	if err != nil {
		log.Printf("Error: failed to scan archive %q: %#v\n", fsys.root, err)
		return
	}

	events.Send(metas)

	for _, meta := range metaSlice {
		if meta.file.Hash != "" {
			continue
		}
		log.Printf("hash %q\n", meta.file.Path)
		meta.file.Hash = fsys.hashFile(meta.file)
		events.Send(fs.FileHashed{
			Path: meta.file.Path,
			Hash: meta.file.Hash,
		})
	}
}

func (fsys *FS) readMeta() map[uint64]*fs.FileMeta {
	metas := map[uint64]*fs.FileMeta{}
	absHashFileName := filepath.Join(fsys.root, hashFileName)
	hashInfoFile, err := os.Open(absHashFileName)
	if err != nil {
		return metas
	}
	defer hashInfoFile.Close()

	records, err := csv.NewReader(hashInfoFile).ReadAll()
	if err != nil || len(records) == 0 {
		return metas
	}

	for _, record := range records[1:] {
		if len(record) == 5 {
			iNode, er1 := strconv.ParseUint(record[0], 10, 64)
			path := record[1]
			size, er2 := strconv.ParseUint(record[2], 10, 64)
			modTime, er3 := time.Parse(time.RFC3339, record[3])
			modTime = modTime.UTC().Round(time.Second)
			hash := record[4]
			if hash == "" || er1 != nil || er2 != nil || er3 != nil {
				continue
			}

			metas[iNode] = &fs.FileMeta{
				Path:    path,
				Size:    int(size),
				ModTime: modTime,
				Hash:    hash,
			}

			info, ok := metas[iNode]
			if hash != "" && ok && info.ModTime == modTime && info.Size == int(size) {
				metas[iNode].Hash = hash
			}
		}
	}
	return metas
}

func (s *FS) storeMeta(root string, metas []*meta) error {
	result := make([][]string, 1, len(metas)+1)
	result[0] = []string{"INode", "Name", "Size", "ModTime", "Hash"}

	for _, meta := range metas {
		if meta.file.Hash == "" {
			continue
		}
		result = append(result, []string{
			fmt.Sprint(meta.inode),
			norm.NFC.String(meta.file.Path),
			fmt.Sprint(meta.file.Size),
			meta.file.ModTime.UTC().Format(time.RFC3339Nano),
			meta.file.Hash,
		})
	}

	absHashFileName := filepath.Join(root, hashFileName)
	hashInfoFile, err := os.Create(absHashFileName)

	if err != nil {
		return err
	}
	err = csv.NewWriter(hashInfoFile).WriteAll(result)
	_ = hashInfoFile.Close()
	return err
}

func (fsys *FS) hashFile(meta *fs.FileMeta) string {
	hash := sha256.New()
	buf := make([]byte, bufSize)
	path := filepath.Join(fsys.root, meta.Path)

	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error: failed to scan archive %q: %#v\n", fsys.root, err)
		return ""
	}
	defer file.Close()

	offset := bufSize
	if meta.Size > 2*bufSize {
		offset = meta.Size - bufSize
	}
	nr, er := file.Read(buf)
	if er != nil && er != io.EOF {
		log.Printf("Error: failed to scan archive %q: %#v\n", fsys.root, err)
		return ""
	}
	hash.Write(buf[0:nr])
	if meta.Size > bufSize {
		nr, er := file.ReadAt(buf, int64(offset))
		if er != nil && er != io.EOF {
			log.Printf("Error: failed to scan archive %q: %#v\n", fsys.root, err)
			return ""
		}
		hash.Write(buf[0:nr])
	}

	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func AbsPath(path string) (string, error) {
	var err error
	path, err = filepath.Abs(path)
	path = norm.NFC.String(path)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(path)
	if err != nil {
		return "", err
	}
	return path, nil
}
