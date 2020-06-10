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

	onWin := strings.Contains(runtime.GOOS, "windows")
	onMac := strings.Contains(runtime.GOOS, "darwin")
	// onLinux := !onWin && !onMac

	copyTo := func(src, dest string) {
		if err := copy.Copy(src, dest); err != nil {
			panic(err)
		}
	}

	// Note that this script is meant to be run from a terminal at the project root.
	// It is specifically not meant to be built into an executable and run by double-clicking in
	// Finder, on Mac OS.

	baseDir := filepath.Join("bin")

	// We always remove any pre-existing platform directory before building to ensure it's fresh.
	if err := os.RemoveAll(baseDir); err != nil {
		panic(err)
	}

	copyTo("changelog.txt", filepath.Join(baseDir, "changelog.txt"))

	if onMac {
		baseDir = filepath.Join("bin", "MasterPlan.app", "Contents", "MacOS")
	}

	// Copy the assets folder to the bin directory

	copyTo("assets", filepath.Join(baseDir, "assets"))

	log.Println("Assets copied.")

	filename := filepath.Join(baseDir, "MasterPlan")

	args := []string{"build", "-o", filename, "./"}

	if onWin {
		filename += ".exe"
		// The -H=windowsgui -ldflag is to make sure Go builds a Windows GUI app so the command prompt doesn't stay
		// open while MasterPlan is running. It has to be only if you're building on Windows because this flag
		// gets passed to the compiler and XCode wouldn't build if on Mac I leave it in there.
		args = []string{"build", "-ldflags", "-H=windowsgui", "-o", filename, "./"}
	}

	log.Println("Building binary...")

	result, err := exec.Command("go", args...).CombinedOutput()

	if string(result) != "" {
		log.Println(string(result))
	}

	// Add the stuff for Mac
	if onMac {
		copyTo(filepath.Join("other_sources", "Info.plist"), filepath.Join("bin", "MasterPlan.app", "Contents", "Info.plist"))
		copyTo(filepath.Join("other_sources", "macicons.icns"), filepath.Join("bin", "MasterPlan.app", "Contents", "Resources", "macicons.icns"))
	}

	// The final executable should be, well, executable for everybody. 777 should do it.
	os.Chmod(filename, 0777)

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
