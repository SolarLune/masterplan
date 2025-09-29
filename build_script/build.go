package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archives"
	"github.com/otiai10/copy"
)

func build(baseDir string, releaseMode string, targetOS, targetArch string) {

	fmt.Println(`< Beginning build to "` + baseDir + `" for ` + targetOS + `. >`)

	forWin := strings.Contains(targetOS, "windows")
	forMac := strings.Contains(targetOS, "darwin")

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

	fmt.Println("<Assets copied.>")

	outputFilepath := filepath.Join(baseDir, "MasterPlan")

	if forWin {

		outputFilepath += ".exe"

		// Copy the resources.syso so the executable has the generated icon and executable properties compiled in.
		// This is done using go generate with goversioninfo downloaded and "// go:generate goversioninfo -64=true" in main.go.
		copyTo(filepath.Join("other_sources", "resource.syso"), filepath.Join("..", "resource.syso"))

		// Copy in the SDL requirements (.dll files)
		filepath.Walk(filepath.Join("other_sources"), func(path string, info fs.FileInfo, err error) error {
			_, filename := filepath.Split(path)
			if filepath.Ext(path) == ".dll" {
				copyTo(path, filepath.Join(baseDir, filename))
			}
			return nil
		})

	}

	// We should compile statically at some point, but it's broken currently, it seems? See: https://github.com/veandco/go-sdl2/issues/507
	// So for the meantime, we'll just build dynamically and bundle the dependencies on Windows and Mac. On Linux, usually users have the dependencies (SDL2, basically) installed already.

	// The below string cross-compiles by setting CC to an 64-bit Windows version of MinGW
	// CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=` + targetOS + ` GOARCH=amd64 CGO_LDFLAGS="-lSDL2 -lSDL2_gfx" go build -tags ` + releaseMode + ` -ldflags "-s -w -H=windowsgui" -o ` + filename + ` ./`

	var c *exec.Cmd
	var err error

	os.Setenv(`GOOS`, targetOS)
	os.Setenv(`GOARCH`, targetArch)

	// cross-compile:
	if forWin && runtime.GOOS == "linux" {
		os.Setenv("CC", "x86_64-w64-mingw32-gcc")
	} else {
		os.Setenv("CC", "gcc")
	}

	// Default building for the current OS
	if forWin {
		// No static flag for Windows because we're redistributing the DLLs for simplicity, and I have yet to figure out where to put the static libraries on Linux to cross-compile successfully.
		// When a command is more than a single word and is separated by spaces, it has to be in a single "space" (i.e. "static " + releaseMode)

		// Also note that I know it's weird that I'm joining the build script directory here because baseDir is already a folder up; this makes it so
		// bin is the running directory
		c = exec.Command(`go`, `build`, `-ldflags`, `-s -w -H windowsgui`, `-tags`, releaseMode, `-o`, outputFilepath, `.`)
	} else {
		c = exec.Command(`go`, `build`, `-ldflags`, `-s -w`, `-tags`, releaseMode, `-o`, outputFilepath, `.`)
	}

	fmt.Println("<Building binary with args: ", c.Args, ".>")

	text, err := c.CombinedOutput()

	if err != nil {
		fmt.Println("<ERROR: ", string(err.Error())+">")
		fmt.Println(string(text))
	}

	// Add the stuff for Mac
	if forMac {
		baseDir = filepath.Clean(filepath.Join(baseDir, ".."))
		copyTo(filepath.Join("other_sources", "Info.plist"), filepath.Join(baseDir, "Info.plist"))
		copyTo(filepath.Join("other_sources", "macicons.icns"), filepath.Join(baseDir, "Resources", "macicons.icns"))
	}

	// The final executable should be, well, executable for everybody. 0777 should do it for Mac and Linux.
	os.Chmod(outputFilepath, 0777)

	if err == nil {
		fmt.Println("<Build complete!>")
		fmt.Println("")
	}

	if forWin {
		// Remove Resources; we don't need it in the root directory anymore after building.
		os.Remove(filepath.Join("resource.syso"))
	}

}

// Compress the build output in bin. This is a separate step to ensure that any dependencies that need to be copied in from build
// services (like Appveyor) can be done after building in the build service's configuration.
func compress() {

	fmt.Println("<Compressing build...>")

	// Archive in .tar.gz because AppVeyor doesn't handle execution bits properly and I don't want to add a ton to the source code
	// just to box the output up into a .tar.gz.

	os.Chdir("./bin") // Switch to the bin folder and then archive the contents

	versions := []string{}

	filepath.Walk(filepath.Clean("."), func(path string, info os.FileInfo, err error) error {

		dirCount := len(strings.Split(path, string(os.PathSeparator)))

		if info.IsDir() && dirCount == 1 && path != "." {

			ending := ".tar.gz"
			if strings.Contains(path, "windows") {
				ending = ".zip"
			}

			versions = append(versions, path, ending)
		}

		return nil

	})

	ctx := context.Background()
	for i := 0; i < len(versions); i += 2 {

		version := versions[i]
		ending := versions[i+1]

		// We want to create separate archives for each version (e.g. release and demo)
		files, err := archives.FilesFromDisk(ctx, nil, map[string]string{
			version: version + ending,
		})

		if err != nil {
			panic(err)
		}

		format := archives.CompressedArchive{
			Compression: archives.Gz{},
			Archival:    archives.Tar{},
		}

		out, err := os.Create(version + ending)
		if err != nil {
			panic(err)
		}

		defer out.Close()

		err = format.Archive(ctx, out, files)
		if err != nil {
			panic(err)
		}

	}

	fmt.Println("<Build successfully compressed!>")

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

	fmt.Println("Builds found:", buildNames)

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

	buildMP := flag.Bool("b", false, "Build MasterPlan into the bin directory.")
	osFlag := flag.String("os", "", "What target OS to build MasterPlan for. Omitting this flag will build MasterPlan for the current operating system.")
	archFlag := flag.String("arch", "", "What target arch to build MasterPlan for (either 'amd64' or 'arm64'). Omitting this flag will build MasterPlan for the current architecture.")
	compressMP := flag.Bool("c", false, "Compress build output.")
	itch := flag.Bool("i", false, "Upload build to itch.io.")

	flag.Parse()

	if *buildMP {
		targetName := runtime.GOOS
		targetArch := runtime.GOARCH
		if *osFlag != "" {
			targetName = *osFlag
		}
		if *archFlag != "" {
			targetArch = *archFlag
		}

		build(filepath.Join("bin", targetName+"-0.9-Release-"+targetArch), "release", targetName, targetArch)
		build(filepath.Join("bin", targetName+"-0.9-Demo-"+targetArch), "demo", targetName, targetArch)
	}
	if *compressMP {
		compress() // Compresses all built binary folders in the ./bin folder
	}
	if *itch {
		publishToItch()
	}

	if !*buildMP && !*compressMP && !*itch {

		fmt.Println(
			"MASTERPLAN BUILD SCRIPT:\n",
			"To use this script, you can use the following arguments:\n",
			"\n",
			"-b to build MasterPlan for the current OS; cross-platform builds aren't fully supported yet. ",
			"\n",
			"Example to build for current OS, current architecture: go run . -b\n",
			"Example to build for AMD64 Windows: go run . -b -os windows -arch amd64 \n",
			"Example to build for ARM64 Mac: go run . -b -os darwin -arch arm64 \n",
			"Example to build for Mac, current arch: go run . -b -os darwin\n",
			"Example to build for Linux: go run . -b -os linux \n",
			"\n",
			"-c to compress the build output, as a .tar.gz file for Linux or Mac, or a .zip file for Windows.\n",
			"\n",
			"-i to publish the bin contents to itch.",
		)

	}

}
