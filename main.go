package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

var (
	cpuProfileEnabled = false

	colorOutputEnabled = os.Getenv("TERM") != "dumb" &&
		(isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))

	programName  = filepath.Base(os.Args[0])
	outputStream = io.Writer(os.Stdout)
	errorStream  = os.Stderr
)

func init() {
	// This application does not use any threads.
	// Limiting GOMAXPROCS seems to have a positive effect on GC performance.
	runtime.GOMAXPROCS(1)
}

func main() {
	// Parse flags with custom code instead of the flag package since there
	// are so few of them.

	var configFile string

	setConfigFile := func(s string) {
		if configFile == "" {
			configFile = s
		} else {
			fatalf("configuration file specified twice: %q, %q\n", configFile, s)
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
				case "-h", "-help", "--help" /* GNU concession */ :
					usage(outputStream)
					exitSuccess()
				default:
					fatalf("unknown command line flag %q\n", arg)
				}
			} else {
				configPath, err := userConfigurationFilePath(arg)
				if err != nil {
					fatalf("unable to resolve file path for configuration %q: %s\n", arg, err)
				}
				setConfigFile(configPath)
			}
		}
	}

	if configState {
		fatalln("missing -config argument")
	}
	if configFile == "" {
		fatalln("no configuration file specified")
	}

	if cpuProfileEnabled {
		f, err := os.Create(fmt.Sprintf("%s-cpu.pprof", programName))
		if err == nil {
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		colorOutputEnabled = true
	}

	prog, err := loadProgram(configFile)
	if err != nil {
		fatalf("failed to read configuration: %s\n", err)
	}

	encoder := textEncoderDummy
	if colorOutputEnabled {
		outputStream = colorable.NewColorableStdout()
		encoder = textEncoderANSI
	}

	bufferedOutputStream := bufio.NewWriter(outputStream)
	if err := processLogStream(os.Stdin, bufferedOutputStream, prog, encoder); err != nil {
		fatalln(err.Error())
	}
}

// maxInputLineLength is the largest input line accepted before the scanner
// reports an error. It bounds the per-line memory use.
const maxInputLineLength = 64 * 1024

// processLogStream reads lines from reader, applies prog to every line and writes
// the rendered result to writer using encoder. The input reader error is
// reported after the scan loop ends.
func processLogStream(reader io.Reader, writer *bufio.Writer, prog *program, encoder textEncoder) error {
	line := newLine()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, maxInputLineLength), maxInputLineLength)
	for scanner.Scan() {
		// The line object and its state objects are reused between each line. The byte
		// slice for the line content itself is uniquely allocated for each line as it is
		// saved in a match history for match comparisons.
		line.init(append([]byte(nil), scanner.Bytes()...))

		if err := line.applyProgram(prog); err != nil {
			return err
		}
		if err := line.output(writer, encoder); err != nil {
			return fmt.Errorf("failed to output line: %w", err)
		}
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("failed to output line: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	return nil
}

// userConfigurationFilePath returns the path of the named user configuration file.
func userConfigurationFilePath(name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "rainbow", name+".rainbow"), nil
}

// usage outputs the program usage message to w.
func usage(w io.Writer) {
	configPath := "<config-dir>/rainbow/CONFIG.rainbow"
	if path, err := userConfigurationFilePath("CONFIG"); err == nil {
		configPath = path
	}

	fmt.Fprintf(w, `Rainbow is a log file colorer that acts as a stream processor. Match and
action rules are applied according to configuration to each line read
from stdin, outputting them to stdout.

USAGE:
  %s [OPTIONS]

OPTIONS:
  -h, -help     Show help message
  -color        Force color for non-TTY output
  -config FILE  Use configuration FILE
  CONFIG        Use configuration from %s

EXAMPLE:
  %s CONFIG < logfile
`, programName, configPath, programName)
}

// fatalf writes a formatted error to the error stream and exits with failure.
func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(errorStream, format, a...)
	exitFail()
}

// fatalln writes an error to the error stream and exits with failure.
func fatalln(a ...interface{}) {
	fmt.Fprintln(errorStream, a...)
	exitFail()
}

// exitSuccess exits the process with a success status.
func exitSuccess() {
	os.Exit(0)
}

// exitFail exits the process with a failure status.
func exitFail() {
	os.Exit(1)
}
