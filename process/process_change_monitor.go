package process

import (
    "fmt"
	"github.com/ochinchina/filechangemonitor"
)


var fileChangeMonitor = filechangemonitor.NewFileChangeMonitor(10)

func AddProgramChangeMonitor(path string, fileChangeCb func(path string, mode filechangemonitor.FileChangeMode) ) {
	fileChangeMonitor.AddMonitorFile(path,
		false,
		filechangemonitor.NewExactFileMatcher(path),
		filechangemonitor.NewFileChangeCallbackWrapper( fileChangeCb ),
		filechangemonitor.NewFileMD5CompareInfo())
}

func AddConfigChangeMonitor( path string, filePattern string, fileChangeCb func(path string, mode filechangemonitor.FileChangeMode) ) {
    fmt.Printf( "filePattern=%s\n", filePattern )
    fileChangeMonitor.AddMonitorFile(path,
        true,
        filechangemonitor.NewPatternFileMatcher( filePattern ),
        filechangemonitor.NewFileChangeCallbackWrapper( fileChangeCb ),
        filechangemonitor.NewFileMD5CompareInfo())
}
