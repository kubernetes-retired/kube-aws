package filegen

import (
	"io/ioutil"
	"os"
	"path"
)

type file struct {
	name string
	data []byte
	mode os.FileMode
}

func File(name string, data []byte, mode os.FileMode) file {
	return file{
		name: name,
		data: data,
		mode: mode,
	}
}

// Render writes all assets to disk.
func Render(files ...file) error {
	for _, file := range files {
		if err := os.MkdirAll(path.Dir(file.name), 0755); err != nil {
			return err
		}

		if err := ioutil.WriteFile(file.name, file.data, file.mode); err != nil {
			return err
		}
	}
	return nil
}
