package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// failingReader returns its data and then fails with err.
type failingReader struct {
	data []byte
	err  error
}

// Read copies the remaining data and reports the configured error once the
// data is exhausted.
func (r *failingReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

// testApplyConfigurationToLog applies a configuration to a log file and prints the
// rendered segments using the test encoder.
func testApplyConfigurationToLog(configPath, logPath string) {
	prog, err := loadProgram(configPath)
	if err != nil {
		fmt.Printf("failed to read config: %s\n", err)
		return
	}

	log, err := os.Open(logPath)
	if err != nil {
		fmt.Printf("failed to open log file: %s\n", err)
		return
	}
	defer log.Close()

	line := newLine()
	scanner := bufio.NewScanner(log)
	for scanner.Scan() {
		// The line object and its state objects are reused between each line. The byte
		// slice for the line content itself is uniquely allocated for each line as it's
		// saved in a match history for match comparisons.
		line.init(append([]byte(nil), scanner.Bytes()...))

		if err = line.applyProgram(prog); err != nil {
			fmt.Println(err.Error())
			return
		}

		if err = line.output(os.Stdout, textEncoderTest); err != nil {
			fatalf("failed to output line: %s\n", err)
			return
		}
	}
}

// Example applies the example configuration to the example log and prints the
// rendered segments using the test encoder.
func Example() {
	testApplyConfigurationToLog("testdata/config/example.rainbow", "testdata/logs/example.log")
	//Output:
	// fg:cyan,bg:none,mod:[]                  {2018-08-25 }
	// fg:cyan,bg:none,mod:[bold]              {12:55:33}
	// fg:cyan,bg:none,mod:[]                  {.123 [DEBUG]  Bob:   movement detected; }
	// fg:cyan,bg:none,mod:[bold]              {sector}
	// fg:cyan,bg:none,mod:[]                  {=X2 }
	// fg:cyan,bg:none,mod:[bold]              {count}
	// fg:cyan,bg:none,mod:[]                  {=3}
	// fg:none,bg:none,mod:[]                  {
	// }
	// fg:none,bg:none,mod:[]                  {2018-08-25 12:55:33.125 [NOTICE] }
	// fg:none,bg:none,mod:[bold]              {Bob}
	// fg:none,bg:none,mod:[]                  {:   informing Fred of movement; }
	// fg:none,bg:none,mod:[bold]              {sector}
	// fg:none,bg:none,mod:[]                  {=X2}
	// fg:none,bg:none,mod:[]                  {
	// }
	// fg:iblack,bg:none,mod:[]                {2018-08-25 }
	// fg:iblack,bg:none,mod:[bold]            {12:55:34}
	// fg:iblack,bg:none,mod:[]                {.001 [INFO]   Fred:  dispatching drones; }
	// fg:iblack,bg:none,mod:[bold]            {targetSector}
	// fg:iblack,bg:none,mod:[]                {=X2}
	// fg:none,bg:none,mod:[]                  {
	// }
	// fg:none,bg:none,mod:[]                  {2018-08-25 12:55:34.001 [}
	// fg:white,bg:red,mod:[bold]              {CRIT}
	// fg:none,bg:none,mod:[]                  {]   }
	// fg:none,bg:none,mod:[bold]              {Drone}
	// fg:none,bg:none,mod:[]                  {: damage detected; }
	// fg:none,bg:none,mod:[bold]              {droneID}
	// fg:none,bg:none,mod:[]                  {=3 }
	// fg:none,bg:none,mod:[bold]              {sensor}
	// fg:none,bg:none,mod:[]                  {=hull/3 }
	// fg:none,bg:none,mod:[bold]              {action}
	// fg:none,bg:none,mod:[]                  {=returnHome}
	// fg:none,bg:none,mod:[]                  {
	// }
	// fg:none,bg:none,mod:[]                  {2018-08-25 12:55:34.002 [}
	// fg:white,bg:red,mod:[bold]              {EMERG}
	// fg:none,bg:none,mod:[]                  {]  }
	// fg:none,bg:none,mod:[bold]              {Drone}
	// fg:none,bg:none,mod:[]                  {: damage detected; }
	// fg:none,bg:none,mod:[bold]              {droneID}
	// fg:none,bg:none,mod:[]                  {=3 }
	// fg:none,bg:none,mod:[bold]              {sensor}
	// fg:none,bg:none,mod:[]                  {=engine/1 }
	// fg:none,bg:none,mod:[bold]              {action}
	// fg:none,bg:none,mod:[]                  {=selfDestruct}
	// fg:none,bg:none,mod:[]                  {
	// }
	// fg:none,bg:none,mod:[]                  {2018-08-25 }
	// fg:none,bg:none,mod:[bold]              {12:55:35}
	// fg:none,bg:none,mod:[]                  {.888 [}
	// fg:black,bg:yellow,mod:[]               {WARN}
	// fg:none,bg:none,mod:[]                  {]   }
	// fg:none,bg:none,mod:[bold]              {Fred}
	// fg:none,bg:none,mod:[]                  {:  lost drone; }
	// fg:none,bg:none,mod:[bold]              {droneID}
	// fg:none,bg:none,mod:[]                  {=3 }
	// fg:none,bg:none,mod:[bold]              {lastPosition}
	// fg:none,bg:none,mod:[]                  {=X2/3:7}
	// fg:none,bg:none,mod:[]                  {
	// }
}

// TestProcessLogStream verifies that processLogStream reads and writes lines and
// reports scanner and reader errors instead of truncating the input silently.
func TestProcessLogStream(t *testing.T) {
	const config = `{
    filter: { name: f regexp: (x) }
    apply: { filters: f }
}`

	newProgram := func(t *testing.T) *program {
		t.Helper()
		prog, err := createProgram(strings.NewReader(config))
		if err != nil {
			t.Fatalf("createProgram: %v", err)
		}
		return prog
	}

	t.Run("ordinary input", func(t *testing.T) {
		var buf bytes.Buffer
		writer := bufio.NewWriter(&buf)
		if err := processLogStream(strings.NewReader("hello\nworld\n"), writer, newProgram(t), textEncoderDummy); err != nil {
			t.Fatalf("processLogStream: %v", err)
		}
		if got, want := buf.String(), "hello\nworld\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("line too long", func(t *testing.T) {
		var buf bytes.Buffer
		writer := bufio.NewWriter(&buf)
		input := strings.Repeat("x", maxInputLineLength+1) + "\n"
		err := processLogStream(strings.NewReader(input), writer, newProgram(t), textEncoderDummy)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "failed to read input") {
			t.Errorf("error = %q, want it to contain %q", err, "failed to read input")
		}
	})

	t.Run("reader error", func(t *testing.T) {
		var buf bytes.Buffer
		writer := bufio.NewWriter(&buf)
		readErr := errors.New("read failure")
		reader := &failingReader{data: []byte("line\n"), err: readErr}
		err := processLogStream(reader, writer, newProgram(t), textEncoderDummy)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !errors.Is(err, readErr) {
			t.Errorf("error = %q, want it to wrap %q", err, readErr)
		}
	})
}
