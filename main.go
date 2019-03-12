package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
)

var (
	cpuProfileEnabled = false

	colorOutputEnabled = os.Getenv("TERM") != "dumb" &&
		(isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))

	outputStream = io.Writer(os.Stdout)
	errorStream  = os.Stderr
)

func init() {
	// This application does not use any threads.
	// Limiting GOMAXPROCS seems to have a positive effect on GC performance.
	runtime.GOMAXPROCS(1)
}

func main() {
	// There is some impedance mismatch between the stdlib flag package and my brain.
	// Parse flags using custom code as there are so few of them.

	var configFile string

	setConfigFile := func(s string) {
		if configFile == "" {
			configFile = s
		} else {
			briefUsage()
			exitFail()
		}
	}

	configState := false
	for _, arg := range os.Args[1:] {
		if configState {
			setConfigFile(arg)
			configState = false
		} else {
			if len(arg) > 0 && arg[0] == '-' {
				switch arg {
				case "-color":
					colorOutputEnabled = true
				case "-config":
					configState = true
				case "-help", "--help" /* GNU concession */ :
					detailedUsage()
					exitSuccess()
				default:
					detailedUsage()
					exitFail()
				}
			} else {
				configDir, err := userConfigDir()
				if err != nil {
					fatalf("unable to find user config directory: %s\n", err)
				}
				setConfigFile(configDir + "/rainbow/" + arg + ".rainbow")
			}
		}
	}

	if configState || configFile == "" {
		briefUsage()
		exitFail()
	}

	if cpuProfileEnabled {
		f, err := os.Create("rainbow-cpu.pprof")
		if err == nil {
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		colorOutputEnabled = true
	}

	prog, err := loadProgram(configFile)
	if err != nil {
		fatalf("failed to read config: %s\n", err)
	}

	encoder := textEncoderDummy
	if colorOutputEnabled {
		outputStream = colorable.NewColorableStdout()
		encoder = textEncoderANSI
	}

	bufferedOutputStream := bufio.NewWriter(outputStream)

	line := newLine()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// The line object and its state objects are reused beteween each line. The byte
		// slice for the line content itself is uniquely allocated for each line as it's
		// saved in a match history for match comparisons.
		line.init(append([]byte(nil), scanner.Bytes()...))

		if err = line.applyProgram(prog); err != nil {
			fatalln(err.Error())
		}

		if err = line.output(bufferedOutputStream, encoder); err == nil {
			err = bufferedOutputStream.Flush()
		}
		if err != nil {
			fatalf("failed to output line: %s\n", err)
		}
	}
}

func detailedUsage() {
	errorStream.Write([]byte(`Rainbow is a log file colorer that act as a stream processor. Match and action
rules are applied according to configuration to each line read from stdin,
outputting them to stdout.

`))
	briefUsage()
}

func briefUsage() {
	errorStream.Write([]byte(`Usage:

    -help         Show help
    -color        Force color output for non-TTY output
    -config FILE  Use config FILE
    CONFIG        Use config from ~/.config/rainbow/CONFIG.rainbow 

Example:

    rainbow config < logfile
`))
}

// TODO(jb): Support for other platforms than Linux.
//
// This is currently Linix centric.
// There is the os.UserCacheDir() but I don't think that is the correct place to put user config files.
func userConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	default:
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/.config"
	}
	return dir, nil
}

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(errorStream, format, a...)
	exitFail()
}

func fatalln(a ...interface{}) {
	fmt.Fprintln(errorStream, a...)
	exitFail()
}

func exitSuccess() {
	os.Exit(0)
}

func exitFail() {
	os.Exit(1)
}
