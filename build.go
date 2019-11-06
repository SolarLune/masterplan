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

	// The bin directory should always have a fresh set of assets

	if err := os.RemoveAll(filepath.Join("bin", "assets")); err != nil {
		panic(err)
	}

	// Copy the assets folder to the bin directory
	if err := copy.Copy("assets", filepath.Join("bin", "assets")); err != nil {
		panic(err)
	}

	if err := copy.Copy(".itch.toml", filepath.Join("bin", ".itch.toml")); err != nil {
		panic(err)
	}

	filename := filepath.Join("bin", "MasterPlan_"+runtime.GOOS+"_"+runtime.GOARCH)

	if strings.Contains(runtime.GOOS, "windows") {
		filename += ".exe"
	}

	args := []string{"build", "-o",
		filename,
		"./src/"}

	if err := exec.Command("go", args...).Run(); err != nil {
		panic(err)
	}

	fmt.Println("Build [ " + filename + " ] complete!")

}
