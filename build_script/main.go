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

func build(baseDir string, releaseMode string, targetOS string) {

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

	filename := filepath.Join(baseDir, "MasterPlan")

	if forWin {

		filename += ".exe"

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

	// We should compile statically at some point, but it's broken currently, it seems? See: https://github.com/veandco/go-sdl2/issues/507
	// So for the meantime, we'll just build dynamically and bundle the dependencies on Windows and Mac. On Linux, usually users have the dependencies (SDL2, basically) installed already.

	// The below string cross-compiles by setting CC to an 64-bit Windows version of MinGW
	// CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=` + targetOS + ` GOARCH=amd64 CGO_LDFLAGS="-lSDL2 -lSDL2_gfx" go build -tags ` + releaseMode + ` -ldflags "-s -w -H=windowsgui" -o ` + filename + ` ./`

	var c *exec.Cmd
	var err error

	os.Setenv("CGO_ENABLED", "1")
	os.Setenv(`GOOS`, targetOS)
	os.Setenv(`GOARCH`, "amd64")
	os.Setenv(`CGO_LDFLAGS`, "-lSDL2 -lSDL2_gfx")

	// cross-compile:
	if forWin && runtime.GOOS == "linux" {
		os.Setenv("CC", "x86_64-w64-mingw32-gcc")
	} else {
		os.Setenv("CC", "gcc")
	}

	// Default building for the current OS
	if forWin {
		c = exec.Command(`go`, `build`, `-ldflags`, `-s -w -H windowsgui`, `-tags`, releaseMode, `-o`, filename, `./`)
	} else {
		c = exec.Command(`go`, `build`, `-ldflags`, `-s -w`, `-tags`, releaseMode, `-o`, filename, `./`)
	}

	fmt.Println("<Building binary with args: ", c.Args, ".>")

	_, err = c.CombinedOutput()

	if err != nil {
		fmt.Println("<ERROR: ", string(err.Error())+">")
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
		fmt.Println("<Build complete!>")
		fmt.Println("")
	}

	if forWin {
		// Remove Resources; we don't need it in the root directory anymore after building.
		os.Remove("resource.syso")
	}

}

func oldBuild(baseDir string, releaseMode bool, targetOS string) {

	fmt.Println(`< Beginning build to "` + baseDir + `" for ` + targetOS + `. >`)

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

	ldFlags := `"`

	if releaseMode {
		ldFlags += `-X main.releaseMode=true`
	} else {
		ldFlags += `-X main.releaseMode=false`
	}

	static := ``

	if forWin {

		filename += ".exe"

		// The -H=windowsgui -ldflag is to make sure Go builds a Windows GUI app so the command prompt doesn't stay
		// open while MasterPlan is running. It has to be only if you're building on Windows because this flag
		// gets passed to the compiler and XCode wouldn't build if on Mac I leave it in there.

		ldFlags += `-H=windowsgui '-extldflags -static'`

		// Copy the resources.syso so the executable has the generated icon and executable properties compiled in.
		// This is done using go generate with goversioninfo downloaded and "// go:generate goversioninfo -64=true" in main.go.
		copyTo(filepath.Join("other_sources", "resource.syso"), "resource.syso")

		// Copy in the SDL requirements (.dll files)
		// filepath.Walk(filepath.Join("other_sources"), func(path string, info fs.FileInfo, err error) error {
		// 	_, filename := filepath.Split(path)
		// 	if filepath.Ext(path) == ".dll" {
		// 		copyTo(path, filepath.Join(baseDir, filename))
		// 	}
		// 	return nil
		// })

		static = `-tags static`

	}

	ldFlags += `"`

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

			// } else {

			// No command for building from some other OSes to Linux.

		}

		// result, err := exec.Command("go", args...).CombinedOutput()

	} else {

		// Default building for the current OS
		c = exec.Command(`env`, `CGO_ENABLED=1`,
			`GOOS=`+targetOS,
			`GOARCH=amd64`,
			// `CGO_LDFLAGS=-lmingw32 -lSDL2 -lSDL2_gfx`,
			`CGO_LDFLAGS=-lSDL2 -lSDL2_gfx`,
			static,
			`go`, `build`, `-o`, filename, `./`)
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

	for i := 0; i < len(versions); i += 2 {
		version := versions[i]
		ending := versions[i+1]
		// We want to create separate archives for each version (e.g. release and demo)
		archiver.Archive([]string{version}, version+ending)
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
	compressMP := flag.Bool("c", false, "Compress build output.")
	itch := flag.Bool("i", false, "Upload build to itch.io.")

	flag.Parse()

	if *buildMP {
		fmt.Println(*osFlag)
		if *osFlag == "all" {
			build(filepath.Join("bin", "linux-0.8-Release-64"), "release", "linux")
			build(filepath.Join("bin", "linux-0.8-Demo-64"), "demo", "linux")
			build(filepath.Join("bin", "windows-0.8-Release-64"), "release", "windows")
			build(filepath.Join("bin", "windows-0.8-Demo-64"), "demo", "windows")
			build(filepath.Join("bin", "macos-0.8-Release-64"), "release", "darwin")
			build(filepath.Join("bin", "macos-0.8-Demo-64"), "demo", "darwin")
		} else if *osFlag != "" {
			targetName := *osFlag
			if strings.Contains(targetName, "darwin") {
				targetName = "macos"
			}
			build(filepath.Join("bin", targetName+"-0.8-Release-64"), "release", *osFlag)
			build(filepath.Join("bin", targetName+"-0.8-Demo-64"), "demo", *osFlag)
		} else {
			targetName := runtime.GOOS
			if strings.Contains(targetName, "darwin") {
				targetName = "macos"
			}
			build(filepath.Join("bin", targetName+"-0.8-Release-64"), "release", runtime.GOOS)
			build(filepath.Join("bin", targetName+"-0.8-Demo-64"), "demo", runtime.GOOS)
		}
		// Demo builds are paused until MasterPlan v0.8 is the main version.
		// build(filepath.Join("bin", fmt.Sprintf("MasterPlan-%s-Demo", target)), "-X main.releaseMode=true -X main.demoMode=DEMO", *targetOS)
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
