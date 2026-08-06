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

// FileWriter writes log bytes into files stored in the designated directory.
// Log file naming convention is prefix + date + ".log".
// When the current log file size reaches logDataSize, a new log file is created
// on the next write.
type FileWriter struct {
	ctx           context.Context
	sysLogger     Logger
	currentFile   *os.File
	logDir        string
	logFilePrefix string
	logDataSize   int
	mu            sync.Mutex
	size          int
}

func (w *FileWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.currentFile.Write(data)
	if err != nil {
		return n, err
	}
	w.size += n
	if w.size > w.logDataSize {
		if rotateErr := w.handleFileSizeExceedsThresholdUnsafe(); rotateErr != nil {
			w.sysLogger.Errorf(w.ctx, "error while rotating log file: %s", rotateErr)
		}
	}
	return n, nil
}

func (w *FileWriter) handleFileSizeExceedsThresholdUnsafe() (err error) {
	newLogFilePath := filepath.Join(w.logDir, fmt.Sprintf("%s-%d.log", w.logFilePrefix, time.Now().UnixNano()))

	newLogFile, err := os.OpenFile(newLogFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	oldFile := w.currentFile
	w.currentFile = newLogFile
	w.size = 0
	if oldFile != nil {
		_ = oldFile.Close()
	}
	return nil
}

func NewFileWriter(logDir string, filePrefix string, logFileSize int) (*FileWriter, error) {
	var (
		file               *os.File
		stat               os.FileInfo
		absPath            string
		mostRecentModTime  = time.Unix(0, 0)
		mostRecentFilePath string
	)

	err := utils.ProcessWithErrors(func() error {
		var err error
		file, err = os.Open(logDir)
		return err
	}, func() error {
		var err error
		stat, err = file.Stat()
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("path %s is not a directory", logDir)
		}
		return nil
	}, func() error {
		var err error
		absPath, err = filepath.Abs(logDir)
		return err
	}, func() error {
		// Find the most recent file under the directory matching the prefix.
		return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasPrefix(info.Name(), filePrefix) {
				return nil
			}
			if info.ModTime().After(mostRecentModTime) {
				mostRecentModTime = info.ModTime()
				mostRecentFilePath = path
			}
			return nil
		})
	}, func() error {
		if mostRecentFilePath == "" {
			mostRecentFilePath = filepath.Join(absPath, fmt.Sprintf("%s-%d.log", filePrefix, time.Now().UnixNano()))
		}
		var err error
		file, err = os.OpenFile(mostRecentFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
		return err
	}, func() error {
		var e error
		stat, e = file.Stat()
		return e
	})
	if err != nil {
		return nil, err
	}

	return &FileWriter{
		ctx:           context.Background(),
		sysLogger:     StdOutLevelLogger(fmt.Sprintf("log-writer-%s", filePrefix)),
		currentFile:   file,
		logDir:        absPath,
		logFilePrefix: filePrefix,
		logDataSize:   logFileSize,
		size:          int(stat.Size()),
	}, nil
}
