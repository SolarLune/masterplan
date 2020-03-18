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

	osName := runtime.GOOS + "_" + runtime.GOARCH
	if strings.Contains(runtime.GOOS, "darwin") {
		osName = "mac_" + runtime.GOARCH
	}

	baseDir := filepath.Join("bin", osName)

	// We always remove any pre-existing platform directory before building to ensure it's fresh.
	if err := os.RemoveAll(baseDir); err != nil {
		panic(err)
	}

	if err := copy.Copy("changelog.txt", filepath.Join(baseDir, "changelog.txt")); err != nil {
		panic(err)
	}

	if strings.Contains(osName, "mac") {
		baseDir = filepath.Join("bin", osName, "MasterPlan.app", "Contents", "MacOS")
	}

	// Copy the assets folder to the bin directory
	if err := copy.Copy("assets", filepath.Join(baseDir, "assets")); err != nil {
		panic(err)
	}

	log.Println("Assets copied.")

	filename := filepath.Join(baseDir, "MasterPlan")

	args := []string{"build", "-o", filename, "./"}

	if strings.Contains(osName, "windows") {
		filename += ".exe"
		// The -H=windowsgui -ldflag is to make sure Go builds a Windows GUI app so the command prompt doesn't stay.
		// open while MasterPlan is running. It has to be only if you're building on Windows because this flag
		// gets passed to the compiler and XCode wouldn't build if on Mac I leave it in there.
		args = []string{"build", "-ldflags", "-H=windowsgui", "-o", filename, "./"}
	}

	log.Println("Building binary...")

	result, err := exec.Command("go", args...).CombinedOutput()

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
