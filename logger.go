package main

import (
	"fmt"
	"runtime"
	"time"
)

// Log logs a message with a given level.
// level: 0-4, 0 is debug, 1 is info, 2 is warning, 3 is error, 4 is critical.
func Log(level int, msg string) {
	output := ""
	// log time info
	output += "" + time.Now().Format("2006-01-02 15:04:05") + " | "

	// log level info
	switch level {
	case 0:
		output += "[DEBUG] "
	case 1:
		output += "[INFO] "
	case 2:
		output += "[WARNING] "
	case 3:
		output += "[ERROR] "
	case 4:
		output += "[CRITICAL] "
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
	fmt.Println(output)
}
