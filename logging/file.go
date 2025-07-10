package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	ctx         context.Context
	sysLogger   Logger
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
	r, err := w.append(data)
	if err == nil {
		// only increment file size if write is successful
		w.incrementSizeAndMaybeSwitchFile(len(data))
	}
	return r, err
}

func (w *FileWriter) incrementSizeAndMaybeSwitchFile(dataSize int) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.size > w.logDataSize {
		if err := w.handleFileSizeExceedsThresholdUnsafe(); err != nil {
			w.sysLogger.Errorf(w.ctx, "error while writing to log file: %s", err)
		}
	}
	w.size += dataSize
}

func (w *FileWriter) append(data []byte) (int, error) {
	// find the offset to the bottom
	offset, err := w.currentFile.Seek(0, 2)
	if err != nil {
		return -1, err
	}
	return w.currentFile.WriteAt(data, offset)
}

func (w *FileWriter) handleFileSizeExceedsThresholdUnsafe() (err error) {
	newLogFilePath := fmt.Sprintf("%s/%s-%s.log", w.logDir, w.logFilePrefix, time.Now().Format(time.RFC3339))

	newLogFile, err := os.Create(newLogFilePath)
	if err != nil {
		return err
	}
	w.currentFile = newLogFile
	err = w.currentFile.Close()
	if err != nil {
		w.sysLogger.Errorf(w.ctx, "Error closing log file: %v", err)
	}
	w.size = 0
	return
}

func NewFileWriter(logDir string, filePrefix string, logFileSize int) (w *FileWriter, err error) {
	var (
		file               *os.File
		stat               os.FileInfo
		absPath            string
		mostRecentModTime  time.Time = time.Unix(0, 0)
		mostRecentFilePath string
	)

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
	}, func() error {
		// find all files under the directory and use the latest file as the current log file
		return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasPrefix(info.Name(), filePrefix) {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if info.ModTime().After(mostRecentModTime) {
				// use this file as the log file
				mostRecentFilePath = absPath + "/" + info.Name()
			}
			return nil
		})
	}, func() error {
		if mostRecentFilePath == "" {
			// create file
			logFilePath := fmt.Sprintf("%s/%s-%s.log", absPath, filePrefix, time.Now().Format(time.RFC3339))
			file, err = os.Create(logFilePath)
			return err
		}
		file, err = os.OpenFile(mostRecentFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		return err
	})
	if err != nil {
		return
	}
	return &FileWriter{
		ctx:           context.Background(),
		sysLogger:     StdOutLevelLogger(fmt.Sprintf("log-writter-%s", filePrefix)),
		currentFile:   file,
		logDir:        absPath,
		logFilePrefix: filePrefix,
		logDataSize:   logFileSize,
		lock:          new(sync.Mutex),
	}, nil
}
