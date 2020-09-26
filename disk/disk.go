package disk

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
)

// Storage represents where we put the objects
type Storage struct {
	dir string
	mu  sync.Mutex

	dataDir  string
	indexDir string
	tmpDir   string
}

// New returns a new Cache
func New(dir string) *Storage {
	dataDir := path.Join(dir, "data")
	indexDir := path.Join(dir, "index")
	tmpDir := path.Join(dir, "tmp")
	os.MkdirAll(dir, 0755)
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(indexDir, 0755)
	os.MkdirAll(tmpDir, 0755)

	return &Storage{
		dir:      dir,
		dataDir:  dataDir,
		indexDir: indexDir,
		tmpDir:   tmpDir,
	}
}

// SaveTarball stores a tarball
func (s *Storage) SaveTarball(name string, gz io.Reader, meta io.Writer) error {
	tmpIndex, err := ioutil.TempFile(s.tmpDir, "index")
	if err != nil {
		return err
	}
	defer tmpIndex.Close()
	defer os.Remove(tmpIndex.Name())

	enc := gob.NewEncoder(tmpIndex)
	tb, err := gzip.NewReader(gz)
	if err != nil {
		return err
	}
	defer tb.Close()
	tarReader := tar.NewReader(tb)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
		// save metadata
		err = enc.Encode(header)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			tmpfile, err := ioutil.TempFile(s.tmpDir, "data")
			if err != nil {
				return err
			}
			hash := sha256.New()
			multiwriter := io.MultiWriter(hash, tmpfile)
			io.Copy(multiwriter, tarReader)
			md5sum := hex.EncodeToString(hash.Sum(nil))
			// store it
			destDir := path.Join(s.dataDir, md5sum[:2])
			os.MkdirAll(destDir, 0755)
			destFile := path.Join(destDir, md5sum)
			// this needs better error checking
			if _, err = os.Stat(destFile); os.IsNotExist(err) {
				os.Rename(tmpfile.Name(), path.Join(destDir, md5sum))
			}
			// it's fine if this fails
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			err = enc.Encode(md5sum)
			if err != nil {
				return err
			}
		default:
			// skip it
		}
	}
	// save the index
	destIndexFile := path.Join(s.indexDir, name)
	_, err = os.Stat(destIndexFile)
	// overwrite if needed
	if err != nil && os.IsExist(err) {
		os.Remove(destIndexFile)
	}
	os.Rename(tmpIndex.Name(), destIndexFile)
	return nil
}

// FetchTarball ...
func (s *Storage) FetchTarball(name string, w io.Writer) error {
	indexFile := path.Join(s.indexDir, name)
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return err
	}
	f, err := os.Open(indexFile)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	tarWriter := tar.NewWriter(gzw)
	defer tarWriter.Close()
	for {
		var header tar.Header
		err := dec.Decode(&header)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		tarWriter.WriteHeader(&header)
		if header.Typeflag != tar.TypeReg {
			continue
		}
		var sha256 string
		err = dec.Decode(&sha256)
		if err != nil {
			return err
		}
		dataFile := path.Join(s.dataDir, sha256[:2], sha256)
		data, err := os.Open(dataFile)
		if err != nil {
			return err
		}
		defer data.Close()
		io.Copy(tarWriter, data)
	}
	return nil
}

// RequestHandler ...
func (s *Storage) RequestHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// strip leading '/'
	filename := r.URL.Path[1:]

	switch m := r.Method; m {
	case http.MethodGet:
		log.Println("Fetching: " + filename)
		w.Header().Set("Content-Type", "application/gzip")
		err := s.FetchTarball(filename, w)
		if err != nil {
			log.Println(err)
		}
	case http.MethodPut:
		log.Println("Stashing: " + filename)
		err := s.SaveTarball(filename, r.Body, ioutil.Discard)
		if err != nil {
			log.Println(err)
		}
	default:
		log.Println("Dunno what to do!")
		http.Error(w, "Dunno what you mean dude", http.StatusMethodNotAllowed)
	}
}
