package helper

import (
	"encoding/binary"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type Tdx struct {
	ext string // file extension
	// Logger is the slog logger instance.
	Logger *slog.Logger
}

func NewTdx() (*Tdx, error) {
	c := &Tdx{
		Logger: slog.Default(),
	}
	return c, nil
}

func (c *Tdx) Read(filePath string) (<-chan Bar, error) {
	file, err := os.Open(filePath)
	if err != nil {
		c.Logger.Error("Unable to open file.", "error", err)
		return nil, err
	}
	ext := filepath.Ext(filePath)
	c.ext = ext
	wg := &sync.WaitGroup{}
	rows := Waitable(wg, c.ReadFromFile(file))

	go func() {
		wg.Wait()
		err := file.Close()
		if err != nil {
			c.Logger.Error("Unable to close file.", "error", err)
		}
	}()

	return rows, nil
}

func (c *Tdx) ReadFromFile(file *os.File) <-chan Bar {
	rows := make(chan Bar)

	go func() {
		defer close(rows)

		fi, err := file.Stat()
		if err != nil {
			c.Logger.Error("Unable to get file size.", "error", err)
			return
		}

		size := fi.Size()
		count := size / 32

		//var barSize uint
		for i := 0; i < int(count); i++ {
			var bar Bar
			switch c.ext {
			case ".day":
				bar = new(dayBar)
				//barSize = 1440
			case ".5":
				bar = new(fiveBar)
				//barSize = 5
			case ".lc5":
				bar = new(lcnBar)
				//barSize = 5
			case ".lc1":
				bar = new(lcnBar)
				//barSize = 1
			default:
				c.Logger.Error("Unsupported file extension.", "ext", c.ext)
				return
			}
			err = binary.Read(file, binary.LittleEndian, bar)
			if err != nil {
				c.Logger.Error("Unable to read binary data.", "error", err)
				return
			}
			rows <- bar
		}
	}()

	return rows
}

func ReadFromTdxFile[T any](fileName string) (<-chan Bar, error) {
	c, err := NewTdx()
	if err != nil {
		return nil, err
	}

	return c.Read(fileName)
}
