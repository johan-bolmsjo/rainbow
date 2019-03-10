package main

import (
	"bufio"
	"fmt"
	"os"
)

func testApplyConfigToLog(configPath, logPath string) {
	prog, err := loadProgram(configPath)
	if err != nil {
		fmt.Printf("failed to read config: %s\n", err)
		return
	}

	log, err := os.Open(logPath)
	if err != nil {
		fmt.Printf("failed to log file: %s\n", err)
		return
	}
	defer log.Close()

	line := newLine()
	scanner := bufio.NewScanner(log)
	for scanner.Scan() {
		// The line object and its state objects are reused beteween each line. The byte
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

func Example_1() {
	testApplyConfigToLog("testdata/config/example.rainbow", "testdata/logs/example.log")
	//Output:
	//fg:cyan,bg:none,mod:[]                  {2018-08-25 }
	//fg:cyan,bg:none,mod:[bold]              {12:55:33}
	//fg:cyan,bg:none,mod:[]                  {.123 [DEBUG]  Bob:   movement detected; }
	//fg:cyan,bg:none,mod:[bold]              {sector}
	//fg:cyan,bg:none,mod:[]                  {=X2 }
	//fg:cyan,bg:none,mod:[bold]              {count}
	//fg:cyan,bg:none,mod:[]                  {=3}
	//fg:none,bg:none,mod:[]                  {
	//}
	//fg:none,bg:none,mod:[]                  {2018-08-25 12:55:33.125 [NOTICE] }
	//fg:none,bg:none,mod:[bold]              {Bob}
	//fg:none,bg:none,mod:[]                  {:   informing Fred of movement; }
	//fg:none,bg:none,mod:[bold]              {sector}
	//fg:none,bg:none,mod:[]                  {=X2}
	//fg:none,bg:none,mod:[]                  {
	//}
	//fg:iblack,bg:none,mod:[]                {2018-08-25 }
	//fg:iblack,bg:none,mod:[bold]            {12:55:34}
	//fg:iblack,bg:none,mod:[]                {.001 [INFO]   Fred:  dispatching drones; }
	//fg:iblack,bg:none,mod:[bold]            {targetSector}
	//fg:iblack,bg:none,mod:[]                {=X2}
	//fg:none,bg:none,mod:[]                  {
	//}
	//fg:none,bg:none,mod:[]                  {2018-08-25 12:55:34.001 [}
	//fg:white,bg:red,mod:[bold]              {CRIT}
	//fg:none,bg:none,mod:[]                  {]   }
	//fg:none,bg:none,mod:[bold]              {Drone}
	//fg:none,bg:none,mod:[]                  {: damage detected; }
	//fg:none,bg:none,mod:[bold]              {droneID}
	//fg:none,bg:none,mod:[]                  {=3 }
	//fg:none,bg:none,mod:[bold]              {sensor}
	//fg:none,bg:none,mod:[]                  {=hull/3 }
	//fg:none,bg:none,mod:[bold]              {action}
	//fg:none,bg:none,mod:[]                  {=returnHome}
	//fg:none,bg:none,mod:[]                  {
	//}
	//fg:none,bg:none,mod:[]                  {2018-08-25 12:55:34.002 [}
	//fg:white,bg:red,mod:[bold]              {EMERG}
	//fg:none,bg:none,mod:[]                  {]  }
	//fg:none,bg:none,mod:[bold]              {Drone}
	//fg:none,bg:none,mod:[]                  {: damage detected; }
	//fg:none,bg:none,mod:[bold]              {droneID}
	//fg:none,bg:none,mod:[]                  {=3 }
	//fg:none,bg:none,mod:[bold]              {sensor}
	//fg:none,bg:none,mod:[]                  {=engine/1 }
	//fg:none,bg:none,mod:[bold]              {action}
	//fg:none,bg:none,mod:[]                  {=selfDestruct}
	//fg:none,bg:none,mod:[]                  {
	//}
	//fg:none,bg:none,mod:[]                  {2018-08-25 }
	//fg:none,bg:none,mod:[bold]              {12:55:35}
	//fg:none,bg:none,mod:[]                  {.888 [}
	//fg:black,bg:yellow,mod:[]               {WARN}
	//fg:none,bg:none,mod:[]                  {]   }
	//fg:none,bg:none,mod:[bold]              {Fred}
	//fg:none,bg:none,mod:[]                  {:  lost drone; }
	//fg:none,bg:none,mod:[bold]              {droneID}
	//fg:none,bg:none,mod:[]                  {=3 }
	//fg:none,bg:none,mod:[bold]              {lastPosition}
	//fg:none,bg:none,mod:[]                  {=X2/3:7}
	//fg:none,bg:none,mod:[]                  {
	//}
}
