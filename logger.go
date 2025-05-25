package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	LoggedFileHandler *os.File
	LastLogFile       string
	SyncThreadCancle  context.CancelFunc
	LogData           struct {
		Debugs uint32
		Infos  uint32
		Warns  uint32
		Errors uint32
		Crits  uint32
	}
)

// Log logs a message with a given level.
// level: 0-4, 0 is debug, 1 is info, 2 is warning, 3 is error, 4 is critical.
func Log(level int, msg string) {
	if SyncThreadCancle == nil {
		var ctx context.Context
		ctx, SyncThreadCancle = context.WithCancel(context.Background())
		go autoSync(ctx)
	}
	output := ""
	// log time info
	output += time.Now().Format("2006-01-02 15:04:05") + " | "

	// log level info
	switch level {
	case 0:
		output += "[DEBUG] "
		atomic.AddUint32(&LogData.Debugs, 1)
	case 1:
		output += "[INFO] "
		atomic.AddUint32(&LogData.Infos, 1)
	case 2:
		output += "[WARNING] "
		atomic.AddUint32(&LogData.Warns, 1)
	case 3:
		output += "[ERROR] "
		atomic.AddUint32(&LogData.Errors, 1)
	case 4:
		output += "[CRITICAL] "
		atomic.AddUint32(&LogData.Crits, 1)
	}
	// check log level
	if level < Config.LoggerCfg.Level {
		return
	}

	// log message
	output += msg

	// log stack trace if level is error or critical
	if level >= 3 {
		output += " | "
		_, file, line, ok := runtime.Caller(1)
		if ok {
			output += "at " + file + ":" + fmt.Sprint(line) + " "
		} else {
			output += "at unknown location "
		}
	}
	if !Config.LoggerCfg.DisableStdout {
		fmt.Println(output)
	}
	if Config.LoggerCfg.LogFile != "" {
		if LastLogFile == Config.LoggerCfg.LogFile {
			LoggedFileHandler.WriteString(output + "\n")
		} else {
			if LoggedFileHandler != nil {
				LoggedFileHandler.Close()
			}
			LoggedFileHandler, _ = os.OpenFile(Config.LoggerCfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
			LoggedFileHandler.WriteString(output + "\n")
		}
		LastLogFile = Config.LoggerCfg.LogFile
	}
}

func autoSync(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if LoggedFileHandler != nil {
				LoggedFileHandler.Sync()
				LoggedFileHandler.Close()
			}
			return
		default:
			if Config.LoggerCfg.LogFile != "" && LoggedFileHandler != nil {
				LoggedFileHandler.Sync()
			}
			if Config.LoggerCfg.FileSyncInterval < 1 {
				Config.LoggerCfg.FileSyncInterval = 1
			}
			time.Sleep(time.Duration(Config.LoggerCfg.FileSyncInterval) * time.Second)
		}
	}
}

func CloseLogger() {
	if SyncThreadCancle != nil {
		SyncThreadCancle()
	}
}
