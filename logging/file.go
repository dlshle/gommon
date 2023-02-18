package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dlshle/gommon/utils"
)

// FileWriter
// write logs into files stored in the designated directory
// log file naming convention is prefix+date_iso_string.log
// when the current log file size reaches logDataSize, a new
// log file will be created for next writes until the file
// size is over logDataSize again
type FileWriter struct {
	currentFile *os.File
	logDir      string

	logFilePrefix string
	logDataSize   int
	lock          *sync.Mutex
	size          int
}

func (w *FileWriter) Write(data []byte) (int, error) {
	return w.write(data)
}

func (w *FileWriter) write(data []byte) (int, error) {
	defer (func() {
		w.size += len(data)
	})()
	if (w.size + len(data)) > w.logDataSize {
		if err := w.handleFileSizeExceedsThreshold(); err != nil {
			// TODO bad error handling
			// TODO write to another file
			panic(err)
		}
	}
	return w.append(data)
}

func (w *FileWriter) append(data []byte) (int, error) {
	// find the offset to the bottom
	offset, err := w.currentFile.Seek(0, 2)
	if err != nil {
		return -1, err
	}
	return w.currentFile.WriteAt(data, offset)
}

// TODO maybe use a buffer when lock is in use?
func (w *FileWriter) handleFileSizeExceedsThreshold() (err error) {
	newLogFilePath := fmt.Sprintf("%s/%s-%s.log", w.logDir, w.logFilePrefix, time.Now().String())
	w.lock.Lock()
	defer w.lock.Unlock()
	err = w.currentFile.Close()
	var newLogFile *os.File
	err = utils.ProcessWithErrors(func() error {
		err = w.currentFile.Close()
		newLogFile, err = os.Create(newLogFilePath)
		return err
	})
	if err != nil {
		return err
	}
	w.currentFile = newLogFile
	w.size = 0
	return
}

func NewFileWriter(logDir string, filePrefix string, logFileSize int) (w *FileWriter, err error) {
	var file *os.File
	var stat os.FileInfo
	var absPath string
	err = utils.ProcessWithErrors(func() error {
		file, err = os.Open(logDir)
		return err
	}, func() error {
		stat, err = file.Stat()
		if !stat.IsDir() {
			return fmt.Errorf("path %s is not a directory", logDir)
		}
		return err
	}, func() error {
		absPath, err = filepath.Abs(logDir)
		return err
	})
	if err != nil {
		return
	}
	return &FileWriter{
		logDir:        absPath,
		logFilePrefix: filePrefix,
		logDataSize:   logFileSize,
		lock:          &sync.Mutex{},
	}, nil
}
