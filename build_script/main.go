package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archiver"
	"github.com/otiai10/copy"
)

func build(baseDir string, releaseMode bool, targetOS string) {

	fmt.Println(`< Beginning build to "/` + baseDir + `" for ` + targetOS + `. >`)

	forWin := strings.Contains(targetOS, "windows")
	forMac := strings.Contains(targetOS, "darwin")
	// forLinux := !forWin && !forMac
	crossbuild := targetOS != runtime.GOOS

	copyTo := func(src, dest string) {
		if err := copy.Copy(src, dest); err != nil {
			panic(err)
		}
	}

	// Note that this script is meant to be run from a terminal at the project root.
	// It is specifically NOT meant to be built into an executable and run by double-clicking in
	// Finder, on Mac OS.

	// We always remove any pre-existing platform directory before building to ensure it's fresh.
	if err := os.RemoveAll(baseDir); err != nil {
		panic(err)
	}

	copyTo("changelog.txt", filepath.Join(baseDir, "changelog.txt"))

	if forMac {
		baseDir = filepath.Join(baseDir, "MasterPlan.app", "Contents", "MacOS")
	}

	// Copy the assets folder to the bin directory

	copyTo("assets", filepath.Join(baseDir, "assets"))

	fmt.Println("Assets copied.")

	filename := filepath.Join(baseDir, "MasterPlan")

	ldFlags := "-X main.releaseMode=false"

	if releaseMode {
		ldFlags = "-X main.releaseMode=true"
	}

	if forWin {

		filename += ".exe"

		// The -H=windowsgui -ldflag is to make sure Go builds a Windows GUI app so the command prompt doesn't stay
		// open while MasterPlan is running. It has to be only if you're building on Windows because this flag
		// gets passed to the compiler and XCode wouldn't build if on Mac I leave it in there.

		ldFlags += " -H=windowsgui"

		// Copy the resources.syso so the executable has the generated icon and executable properties compiled in.
		// This is done using go generate with goversioninfo downloaded and "// go:generate goversioninfo -64=true" in main.go.
		copyTo(filepath.Join("other_sources", "resource.syso"), "resource.syso")

		// Copy in the SDL requirements (.dll files)
		filepath.Walk(filepath.Join("other_sources"), func(path string, info fs.FileInfo, err error) error {
			_, filename := filepath.Split(path)
			if filepath.Ext(path) == ".dll" {
				copyTo(path, filepath.Join(baseDir, filename))
			}
			return nil
		})

	}

	var c *exec.Cmd
	var err error

	// Basic crossbuilding from Linux to Windows or Mac
	if crossbuild {

		if forWin {

			c = exec.Command(`env`, `CGO_ENABLED=1`,
				`CC=/usr/bin/x86_64-w64-mingw32-gcc`,
				`GOOS=windows`,
				`GOARCH=amd64`,
				`CGO_LDFLAGS=-lmingw32 -lSDL2 -lSDL2_gfx`,
				`CGO_CFLAGS=-pthread`,
				`go`, `build`, `-ldflags`, ldFlags, `-o`, filename, `./`)

		} else if forMac {

			c = exec.Command(
				`env`,
				`CGO_ENABLED=1`,
				`CC=x86_64-apple-darwin20.4-clang`,
				`GOOS=darwin`,
				`GOARCH=amd64`,
				`go`,
				`build`,
				`-tags`, `static`,
				`-ldflags`, `-s -w -a `+ldFlags,
				`-o`, filename, `./`,
			)

		} else {

			// No command for building from some other OSes to Linux.

		}

		// result, err := exec.Command("go", args...).CombinedOutput()

	} else {

		// Default building for the current OS
		c = exec.Command(
			`env`,
			`GOOS=`+targetOS,
			`GOARCH=amd64`,
			`go`, `build`, `-ldflags`, ldFlags, `-o`, filename, `./`)

	}

	fmt.Println("Building binary with args: ", c.Args, " ...")

	_, err = c.CombinedOutput()

	// result, err := exec.Command("go", args...).CombinedOutput()

	if err != nil {
		fmt.Println("ERROR: ", string(err.Error()))
	}

	// Add the stuff for Mac
	if forMac {
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

	if forWin {
		// Remove Resources; we don't need it in the root directory anymore after building.
		os.Remove("resource.syso")
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

	filepath.Walk(filepath.Join("./", "bin"), func(path string, info os.FileInfo, err error) error {

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
			fmt.Println(string(result), string(err.Error()))
		}

	}

}

func main() {

	buildMP := flag.String("b", "", "Build MasterPlan into the bin directory.")
	compressMP := flag.Bool("c", false, "Compress build output.")
	itch := flag.Bool("i", false, "Upload build to itch.io.")

	flag.Parse()

	if *buildMP != "" {
		if *buildMP == "all" {
			build(filepath.Join("bin", "linux-amd64-0.8-Release"), true, "linux")
			build(filepath.Join("bin", "windows-amd64-0.8-Release"), true, "windows")
			build(filepath.Join("bin", "macos-amd64-0.8-Release"), true, "darwin")
		} else {
			target := strings.ReplaceAll(*buildMP, "/", "-")
			build(filepath.Join("bin", target+"-amd64-0.8-Release"), true, target)
		}
		// Demo builds are paused until MasterPlan v0.8 is the main version.
		// build(filepath.Join("bin", fmt.Sprintf("MasterPlan-%s-Demo", target)), "-X main.releaseMode=true -X main.demoMode=DEMO", *targetOS)
	}
	if *compressMP {
		compress()
	}
	if *itch {
		publishToItch()
	}

	if *buildMP == "" && !*compressMP && !*itch {

		fmt.Println(
			"MASTERPLAN BUILD SCRIPT:\n",
			"To use this script, you can use the following arguments:\n",
			"\n",
			"-b to build MasterPlan for the current OS. If you're on Linux, you can cross-compile to Windows or Mac by specifying the target OS name.\n",
			"Example to build for Mac: >go run ./build_script/main.go -b -os darwin/amd64 \n",
			"Example to build for Windows: >go run ./build_script/main.go -b -os windows/amd64 \n",
			"Example to build for Linux: >go run ./build_script/main.go -b -os linux/amd64 \n",
			"\n",
			"Passing all for the OS name (e.g. -b all) creates a 64-bit build for all operating systems.\n",
			"\n",
			"-c to compress the build output, as a .tar.gz file for Linux or Mac, or a .zip file for Windows.\n",
			"\n",
			"-i to publish the bin contents to itch.",
		)

	}

}
