package disk

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
)

const testDataDir = "testdata"

var testStorage *Storage

func setup() {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "disk-test-*")
	if err != nil {
		log.Fatalln(err.Error())
	}
	testStorage = New(tmpDir)
}

func shutdown() {
	os.RemoveAll(testStorage.dir)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestDiskDirs(t *testing.T) {
	if _, err := os.Stat(testStorage.dir); os.IsNotExist(err) {
		t.Error("Did not create dir")
	}
	if _, err := os.Stat(testStorage.dataDir); os.IsNotExist(err) {
		t.Error("Did not create data dir")
	}
	if _, err := os.Stat(testStorage.indexDir); os.IsNotExist(err) {
		t.Error("Did not create index dir")
	}
	if _, err := os.Stat(testStorage.tmpDir); os.IsNotExist(err) {
		t.Error("Did not create test dir")
	}
}

func TestBasicFlow(t *testing.T) {
	testTarball := "tbdd.tar.gz"
	tarballPath := path.Join(testDataDir, testTarball)
	f, err := os.Open(tarballPath)
	if err != nil {
		t.Error(err.Error())
	}
	if err := testStorage.PutTarball(testTarball, f); err != nil {
		t.Error(err.Error())
	}
	f.Close()

	// now extract it
	tmpTarball, err := ioutil.TempFile(os.TempDir(), "out-*.tar.gz")
	if err != nil {
		t.Error(err.Error())
	}
	if err := testStorage.GetTarball(testTarball, tmpTarball); err != nil {
		t.Error(err.Error())
	}
	tmpTarball.Close()
}
