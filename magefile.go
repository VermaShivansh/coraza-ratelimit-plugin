//go:build mage
// +build mage

package main

import (
	"log"

	"github.com/magefile/mage/sh"
	_ "github.com/vermaShivansh/coraza-ratelimit-plugin/plugin"
)

// Test runs all tests.
func Test() error {
	// remove go test cache
	log.Println("Removing test cache")
	if err := sh.RunV("go", "clean", "--testcache"); err != nil {
		return err
	}

	log.Println("Logic Testing")
	if err := sh.RunV("go", "test", "-run", "^TestLogicOfRateLimit$", "./plugin", "-v"); err != nil {
		return err
	}

	log.Println("Stress testing...")
	if err := sh.RunV("go", "test", "-run", "^TestStressOfRateLimit$", "./plugin", "-v"); err != nil {
		return err
	}

	return nil
}

// remove tmp files
func Clean() error {
	if err := sh.RunV("rm", "-rf", "tmp"); err != nil {
		return err
	}

	return nil
}
