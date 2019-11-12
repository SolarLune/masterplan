package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/otiai10/copy"
)

func main() {

	// Note that this script is meant to be run from a terminal at the project root.
	// It is specifically not meant to be built into an executable and run by double-clicking in
	// Finder, on Mac OS.

	// The bin directory should always have a fresh set of assets

	if err := os.RemoveAll(filepath.Join("bin", "assets")); err != nil {
		panic(err)
	}

	// Copy the assets folder to the bin directory
	if err := copy.Copy("assets", filepath.Join("bin", "assets")); err != nil {
		panic(err)
	}

	fmt.Println("Assets copied...")

	if err := copy.Copy(".itch.toml", filepath.Join("bin", ".itch.toml")); err != nil {
		panic(err)
	}

	fmt.Println(".itch.toml copied...")

	filename := filepath.Join("bin", "MasterPlan_"+runtime.GOOS+"_"+runtime.GOARCH)

	if strings.Contains(runtime.GOOS, "windows") {
		filename += ".exe"
	}

	fmt.Println("Building binary...")

	args := []string{"build", "-o",
		filename,
		"./"}

	if err := exec.Command("go", args...).Run(); err != nil {
		panic(err)
	}

	fmt.Println("Build [ " + filename + " ] complete!")

}
