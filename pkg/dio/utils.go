package dio

import (
	"fmt"
	"os"
)

func AssertError(w DataWriter, err error, debug bool, override ...string) {
	// Pass, if no error
	if err == nil {
		return
	}
	// Override the error message if provided
	if len(override) > 0 {
		err = fmt.Errorf(override[0], err)
	}
	// If the writer does not implement ErrorWriter, or debug mode enabled, panic with the error
	ew, implements := w.(ErrorWriter)
	if !implements || debug {
		panic(err)
	}
	// Write the error
	ew.WriteError(err)
	// Exit with non-zero code to indicate an error occurred
	os.Exit(1)
}

func AssertWarning(w DataWriter, err error, debug bool, nowarn bool, override ...string) {
	// Pass, if no error or warnings are disabled
	if err == nil || nowarn {
		return
	}
	// Override the warning message if provided
	if len(override) > 0 {
		err = fmt.Errorf(override[0], err)
	}
	// If the writer does not implement WarningWriter, or debug mode enabled, panic with the warning
	ww, implements := w.(WarningWriter)
	if !implements || debug {
		panic(err)
	}
	// Write the warning
	ww.WriteWarning(err)
}
