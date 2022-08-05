package testutils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ava-labs/avalanchego/utils/perms"
	"github.com/stretchr/testify/assert"
)

func CreateZip(assert *assert.Assertions, src string, dest string) {
	zipf, err := os.Create(dest)
	assert.NoError(err)
	defer zipf.Close()

	zipWriter := zip.NewWriter(zipf)
	defer zipWriter.Close()

	// 2. Go through all the files of the source
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(src), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})

	assert.NoError(err)
}

func CreateTarGz(assert *assert.Assertions, src string, dest string, includeTopLevel bool) {
	tgz, err := os.Create(dest)
	assert.NoError(err)
	defer tgz.Close()

	gw := gzip.NewWriter(tgz)
	defer gw.Close()

	tarball := tar.NewWriter(gw)
	defer tarball.Close()

	info, err := os.Stat(src)
	assert.NoError(err)

	var baseDir string
	if includeTopLevel && info.IsDir() {
		baseDir = filepath.Base(src)
	} else {
		baseDir = ""
	}

	err = filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, src))
			}

			fmt.Println("Base dir", baseDir)
			fmt.Println("Header:", header.Name)
			if strings.TrimSuffix(header.Name, "/") == filepath.Base(src) {
				fmt.Println("Hit condition")
				return nil
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			defer func() {
				err := file.Close()
				assert.NoError(err)
			}()
			_, err = io.Copy(tarball, file)
			return err
		})
	assert.NoError(err)
}

func CreateTestArchivePath(t *testing.T, assert *assert.Assertions) (string, func(string)) {
	// create root test dir, will be cleaned up after test
	testDir := t.TempDir()

	// create some test dirs
	dir1 := filepath.Join(testDir, "dir1")
	dir2 := filepath.Join(testDir, "dir2")
	err := os.Mkdir(dir1, perms.ReadWriteExecute)
	assert.NoError(err)
	err = os.Mkdir(dir2, perms.ReadWriteExecute)
	assert.NoError(err)

	// create some (empty) files
	_, err = os.Create(filepath.Join(dir1, "gzipTest11"))
	assert.NoError(err)
	_, err = os.Create(filepath.Join(dir1, "gzipTest12"))
	assert.NoError(err)
	_, err = os.Create(filepath.Join(dir1, "gzipTest13"))
	assert.NoError(err)
	_, err = os.Create(filepath.Join(dir2, "gzipTest21"))
	assert.NoError(err)
	_, err = os.Create(filepath.Join(testDir, "gzipTest0"))
	assert.NoError(err)

	// also create a binary file
	buf := make([]byte, 32)
	_, err = rand.Read(buf)
	assert.NoError(err)
	binFile := filepath.Join(testDir, "binary-test-file")
	err = os.WriteFile(binFile, buf, perms.ReadWrite)
	assert.NoError(err)

	// make sure the same stuff exists
	checkFunc := func(controlDir string) {
		assert.DirExists(filepath.Join(controlDir, "dir1"))
		assert.DirExists(filepath.Join(controlDir, "dir2"))
		assert.FileExists(filepath.Join(controlDir, "dir1", "gzipTest11"))
		assert.FileExists(filepath.Join(controlDir, "dir1", "gzipTest12"))
		assert.FileExists(filepath.Join(controlDir, "dir1", "gzipTest13"))
		assert.FileExists(filepath.Join(controlDir, "dir2", "gzipTest21"))
		assert.FileExists(filepath.Join(controlDir, "gzipTest0"))
		checkBin, err := os.ReadFile(binFile)
		assert.NoError(err)
		assert.Equal(checkBin, buf)
	}

	return testDir, checkFunc
}
