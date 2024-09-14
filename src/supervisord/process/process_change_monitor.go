package process

import (
	"fmt"

	"github.com/ochinchina/filechangemonitor"
)

var fileChangeMonitor = filechangemonitor.NewFileChangeMonitor(10)

// AddProgramChangeMonitor adds program change listener to monitor if the program binary
func AddProgramChangeMonitor(path string, fileChangeCb func(path string, mode filechangemonitor.FileChangeMode)) {
	fileChangeMonitor.AddMonitorFile(path,
		false,
		filechangemonitor.NewExactFileMatcher(path),
		filechangemonitor.NewFileChangeCallbackWrapper(fileChangeCb),
		filechangemonitor.NewFileMD5CompareInfo())
}

// AddConfigChangeMonitor adds program change listener to monitor if any of its configuration files is changed
func AddConfigChangeMonitor(path string, filePattern string, fileChangeCb func(path string, mode filechangemonitor.FileChangeMode)) {
	fmt.Printf("filePattern=%s\n", filePattern)
	fileChangeMonitor.AddMonitorFile(path,
		true,
		filechangemonitor.NewPatternFileMatcher(filePattern),
		filechangemonitor.NewFileChangeCallbackWrapper(fileChangeCb),
		filechangemonitor.NewFileMD5CompareInfo())
}
