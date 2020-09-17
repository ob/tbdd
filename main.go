package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

const dataDir = "storeDir/dataDir"
const indexDir = "storeDir/indexDir"
const tempDir = "storeDir/tempDir"

func saveTarball(name string, gz io.Reader, meta io.Writer) error {
	tmpIndex, err := ioutil.TempFile(tempDir, "index")
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
			tmpfile, err := ioutil.TempFile(tempDir, "data")
			if err != nil {
				return err
			}
			hash := sha256.New()
			multiwriter := io.MultiWriter(hash, tmpfile)
			io.Copy(multiwriter, tarReader)
			md5sum := hex.EncodeToString(hash.Sum(nil))
			// store it
			destDir := path.Join(dataDir, md5sum[:2])
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
	destIndexFile := path.Join(indexDir, name)
	_, err = os.Stat(destIndexFile)
	// overwrite if needed
	if err != nil && os.IsExist(err) {
		os.Remove(destIndexFile)
	}
	os.Rename(tmpIndex.Name(), destIndexFile)
	return nil
}

func fetchTarBall(name string, w io.Writer) error {
	indexFile := path.Join(indexDir, name)
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
		dataFile := path.Join(dataDir, sha256[:2], sha256)
		data, err := os.Open(dataFile)
		if err != nil {
			return err
		}
		defer data.Close()
		io.Copy(tarWriter, data)
	}
	return nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// strip leading '/'
	filename := r.URL.Path[1:]

	switch m := r.Method; m {
	case http.MethodGet:
		log.Println("Fetching: " + filename)
		w.Header().Set("Content-Type", "application/gzip")
		err := fetchTarBall(filename, w)
		if err != nil {
			log.Println(err)
		}
	case http.MethodPut:
		log.Println("Stashing: " + filename)
		err := saveTarball(filename, r.Body, ioutil.Discard)
		if err != nil {
			log.Println(err)
		}
	default:
		log.Println("Dunno what to do!")
		http.Error(w, "Dunno what you mean dude", http.StatusMethodNotAllowed)
	}
}

func main() {
	var port string
	var bindAddr string
	flag.StringVar(&bindAddr, "host", "0.0.0.0", "address to listen on")
	flag.StringVar(&port, "port", "12625", "port on which to listen to")
	flag.Parse()

	// create dirs
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(tempDir, 0755)
	os.MkdirAll(indexDir, 0755)
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:         bindAddr + ":" + port,
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	mux.HandleFunc("/", handle)
	log.Printf("Server listening on http://%s\n", httpServer.Addr)
	httpServer.ListenAndServe()
}
