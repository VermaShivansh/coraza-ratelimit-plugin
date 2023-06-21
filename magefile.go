//go:build mage
// +build mage

package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/magefile/mage/sh"
	_ "github.com/vermaShivansh/coraza-ratelimit-plugin/plugin"
)

var golangCILintVer = "v1.53.3" // https://github.com/golangci/golangci-lint/releases
var errRunGoModTidy = errors.New("go.mod/sum not formatted, commit changes")

// Lint verifies code quality.
func Lint() error {
	log.Println("Lint checks...")
	if err := sh.RunV("go", "run", fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", golangCILintVer), "run"); err != nil {
		return err
	}

	log.Println("Cleaning packages")
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}

	if sh.Run("git", "diff", "--exit-code", "go.mod", "go.sum") != nil {
		return errRunGoModTidy
	}

	return nil
}

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
