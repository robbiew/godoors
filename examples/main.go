package main

import (
	"flag"
	"fmt"
	"os"

	gd "github.com/robbiew/godoors"
)

func main() {

	pathPtr := flag.String("path", "", "path to door32.sys file")

	required := []string{"path"}
	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			// or possibly use `log.Fatalf` instead of:
			fmt.Fprintf(os.Stderr, "missing path to door32.sys directory: -%s \n", req)
			os.Exit(2) // the same exit code flag.Parse uses

		}
	}

	path := *pathPtr

	dropAlias, timeInt, emuInt := gd.DropFileData(path)

	var emuName string

	if emuInt == 1 {
		emuName = "ANSI"
	} else if emuInt == 0 {
		emuName = "ASCII"
	}
	fmt.Println("DROP FILE TEST")
	fmt.Println("Alias: " + dropAlias)
	fmt.Fprintf(os.Stdout, "Emulation: %v\r\n", emuName)
	fmt.Fprintf(os.Stdout, "Time Left: %v\r\n", timeInt)
}
