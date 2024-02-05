package e2e

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
)

func parseArguments() error {
	var err error
	flag.Parse()
	return err
}

func setup() error {
	fmt.Println("Setting up environment")

	// build applications
	err := buildApplications()
	if err != nil {
		return err
	}

	return err
}

// runs E2E tests
func TestMain(m *testing.M) {

	if err := parseArguments(); err != nil {
		flag.Usage()
		log.Fatalf("Error parsing command arguments %s\n", err)
	}

	err := setup()
	if err != nil {
		fmt.Println("Failed setting up environment: ", err)
		os.Exit(-1)
	}

	rc := m.Run()

	os.Exit(rc)
}
