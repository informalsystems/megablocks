package e2e

import (
	"fmt"
	"os"
	"os/exec"
)

// startApplications starts all chain applications
// returns list of cmd pointers chain app processes
func startApplications() ([]*exec.Cmd, error) {
	apps := []*exec.Cmd{}
	cmd, err := startKVStore()
	if err == nil {
		apps = append(apps, cmd)
	}
	return apps, err
}

// stopApplications stops all provided apps
func stopApplications(apps []*exec.Cmd) error {
	fmt.Println("stopping applications")
	var err error

	for _, app := range apps {
		// try a graceful termination of the process
		app.Process.Signal(os.Interrupt)
		rc := app.Wait()
		if rc != nil {
			err = rc
			fmt.Println("error stopping ", app, err)
			app.Process.Kill()
		}
	}
	return err
}

// buildApplications triggers build of all executables
// controlled by Makefile in project root
func buildApplications() error {
	fmt.Println("building applications")
	cmd := exec.Command("make", "build")
	cmd.Dir = "../../"
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error building applications: %s", string(out))
	}
	return err
}
