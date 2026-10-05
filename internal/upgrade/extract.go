package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func unpackAx(data []byte) ([]byte, error) {
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		return unzipNamed(data, "ax")
	}
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return untarNamed(data, "ax")
	}
	return data, nil
}

func unzipNamed(data []byte, want string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name != want && name != want+".exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		return body, err
	}
	return nil, fmt.Errorf("zip has no %s", want)
}

func untarNamed(data []byte, want string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("tar has no %s", want)
		}
		if err != nil {
			return nil, err
		}
		name := filepath.Base(hdr.Name)
		if name != want && !strings.HasPrefix(name, want+".") {
			continue
		}
		return io.ReadAll(tr)
	}
}
