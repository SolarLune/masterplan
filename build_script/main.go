package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archiver"
	"github.com/otiai10/copy"
)

func build(baseDir string, ldFlags string) {

	fmt.Println(fmt.Sprintf("Beginning build to %s.", baseDir))

	onWin := strings.Contains(runtime.GOOS, "windows")
	onMac := strings.Contains(runtime.GOOS, "darwin")

	copyTo := func(src, dest string) {
		if err := copy.Copy(src, dest); err != nil {
			panic(err)
		}
	}

	// Note that this script is meant to be run from a terminal at the project root.
	// It is specifically not meant to be built into an executable and run by double-clicking in
	// Finder, on Mac OS.

	// We always remove any pre-existing platform directory before building to ensure it's fresh.
	if err := os.RemoveAll(baseDir); err != nil {
		panic(err)
	}

	copyTo("changelog.txt", filepath.Join(baseDir, "changelog.txt"))

	if onMac {
		baseDir = filepath.Join(baseDir, "MasterPlan.app", "Contents", "MacOS")
	}

	// Copy the assets folder to the bin directory

	copyTo("assets", filepath.Join(baseDir, "assets"))

	fmt.Println("Assets copied.")

	filename := filepath.Join(baseDir, "MasterPlan")

	if onWin {

		// Copy the resources.syso so the executable has the generated icon and executable properties compiled in.
		// This is done using go generate with goversioninfo downloaded and "// go:generate goversioninfo -64=true" in main.go.
		copyTo(filepath.Join("other_sources", "resource.syso"), "resource.syso")

		filename += ".exe"
		// The -H=windowsgui -ldflag is to make sure Go builds a Windows GUI app so the command prompt doesn't stay
		// open while MasterPlan is running. It has to be only if you're building on Windows because this flag
		// gets passed to the compiler and XCode wouldn't build if on Mac I leave it in there.
		ldFlags += " -H=windowsgui"
	}

	args := []string{"build", "-ldflags", ldFlags, "-o", filename, "./"}

	fmt.Println(fmt.Sprintf("Building binary with flags %s...", args))

	result, err := exec.Command("go", args...).CombinedOutput()

	if string(result) != "" {
		fmt.Println(string(result))
	}

	// Add the stuff for Mac
	if onMac {
		baseDir = filepath.Clean(filepath.Join(baseDir, ".."))
		copyTo(filepath.Join("other_sources", "Info.plist"), filepath.Join(baseDir, "Info.plist"))
		copyTo(filepath.Join("other_sources", "macicons.icns"), filepath.Join(baseDir, "Resources", "macicons.icns"))
	}

	// The final executable should be, well, executable for everybody. 0777 should do it for Mac and Linux.
	os.Chmod(filename, 0777)

	if err == nil {
		fmt.Println("Build complete!")
		fmt.Println("")
	}

	if onWin {
		// Remove Resources; we don't need it in the root directory anymore after building.
		os.Remove("resources.syso")
	}

}

// Compress the build output in bin. This is a separate step to ensure that any dependencies that need to be copied in from build
// services (like Appveyor) can be done after building in the build service's configuration.
func compress() {

	onWin := strings.Contains(runtime.GOOS, "windows")
	ending := ".tar.gz"
	if onWin {
		ending = ".zip"
	}

	// Archive in .tar.gz because AppVeyor doesn't handle execution bits properly and I don't want to add a ton to the source code
	// just to box the output up into a .tar.gz.

	os.Chdir("./bin") // Switch to the bin folder and then archive the contents

	versions := []string{}

	filepath.Walk(filepath.Clean("."), func(path string, info os.FileInfo, err error) error {

		dirCount := len(strings.Split(path, string(os.PathSeparator)))

		if info.IsDir() && dirCount == 1 && path != "." {
			versions = append(versions, path)
		}

		return nil

	})

	for _, version := range versions {
		// We want to create separate archives for each version (e.g. release and demo)
		archiver.Archive([]string{version}, version+ending)
	}

	fmt.Println("Build successfully compressed!")

}

func publishToItch() {

	buildNames := []string{}

	filepath.Walk(filepath.Join("./", "build_script"), func(path string, info os.FileInfo, err error) error {

		dirCount := len(strings.Split(path, string(os.PathSeparator)))

		if info.IsDir() && dirCount == 2 {
			buildNames = append(buildNames, path) // We want to upload the build directories
		}

		return nil

	})

	for _, build := range buildNames {

		buildName := strings.Split(build, string(os.PathSeparator))[1]

		result, err := exec.Command("butler", "push", build, "solarlune/masterplan:"+buildName).CombinedOutput()

		if err == nil {
			fmt.Println("Published", build, "to itch!")
		} else {
			fmt.Println(string(result))
		}

	}

}

func main() {

	buildMP := flag.Bool("b", false, "Build MasterPlan into the bin directory.")
	compressMP := flag.Bool("c", false, "Compress build output.")
	itch := flag.Bool("i", false, "Upload build to itch.io.")
	flag.Parse()

	if *buildMP {
		build(filepath.Join("bin", fmt.Sprintf("MasterPlan-%s-Release", getOSName())), "-X main.releaseMode=true")
		build(filepath.Join("bin", fmt.Sprintf("MasterPlan-%s-Demo", getOSName())), "-X main.releaseMode=true -X main.demoMode=DEMO")
	}
	if *compressMP {
		compress()
	}
	if *itch {
		publishToItch()
	}

	if !*buildMP && !*compressMP && !*itch {

		fmt.Println(
			"To use this script, you can use the following arguments:\n",
			"-b to build MasterPlan for the current OS.\n",
			"-c to compress the build output, as a .tar.gz file for Linux or Mac, or a .zip file for Windows.\n",
			"-i to publish the bin contents to itch.",
		)

	}

}

func getOSName() string {

	platformName := strings.Title(runtime.GOOS)
	if strings.Contains(runtime.GOOS, "darwin") {
		platformName = "Mac"
	}

	return platformName

}
