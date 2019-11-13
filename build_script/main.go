package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/otiai10/copy"
)

func build() {

	// Note that this script is meant to be run from a terminal at the project root.
	// It is specifically not meant to be built into an executable and run by double-clicking in
	// Finder, on Mac OS.

	// The bin directory should always have a fresh set of assets

	if err := os.RemoveAll(filepath.Join("bin", "assets")); err != nil {
		panic(err)
	}

	osName := runtime.GOOS + "_" + runtime.GOARCH
	if strings.Contains(runtime.GOOS, "darwin") {
		osName = "mac_" + runtime.GOARCH
	}

	// Copy the assets folder to the bin directory
	if err := copy.Copy("assets", filepath.Join("bin", osName, "assets")); err != nil {
		panic(err)
	}

	log.Println("Assets copied...")

	// if err := copy.Copy(".itch.toml", filepath.Join("bin", ".itch.toml")); err != nil {
	// 	panic(err)
	// }

	// log.Println(".itch.toml copied...")

	// Build the binary and plop it where the binary should go for this OS
	filename := filepath.Join("bin", osName, "MasterPlan")

	if strings.Contains(osName, "windows") {
		filename += ".exe"
	}

	log.Println("Building binary...")

	result, err := exec.Command("go", "build", "-o", filename, "./").CombinedOutput()

	if string(result) != "" {
		log.Println(string(result))
	}

	if err == nil {
		log.Println("Build [ " + filename + " ] complete!")
	}

}

func publishToItch() {

	// result, err := exec.Command("butler", "push", "solarlune/masterplan").CombinedOutput()
	buildNames := []string{}

	filepath.Walk(filepath.Join("bin"), func(path string, info os.FileInfo, err error) error {

		directoryPath := strings.Split(path, string(filepath.Separator))

		if !info.IsDir() || len(directoryPath) != 2 {
			return nil
		}

		buildNames = append(buildNames, path)

		return nil

	})

	for _, build := range buildNames {

		result, err := exec.Command("butler", "push", build, "solarlune/masterplan:"+build).CombinedOutput()

		if err == nil {
			log.Println("Published", build, "to itch!")
		} else {
			log.Println(string(result))
		}

	}

}

func main() {

	printHelp := func() {
		log.Println("To use this script, use the following arguments:", "-b to build the program for the current OS.", "-i to publish the bin contents to itch.")
	}

	if len(os.Args) != 2 {
		printHelp()
		return
	}

	arg := os.Args[1]

	if strings.HasPrefix(arg, "-b") {
		build()
	} else if strings.HasPrefix(arg, "-i") {
		publishToItch()
	} else {
		log.Println("Error: Command not recognized.")
		printHelp()
		return
	}

}
