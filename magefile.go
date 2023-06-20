//go:build mage
// +build mage

package main

import (
	"log"

	"github.com/magefile/mage/sh"
	_ "github.com/vermaShivansh/coraza-ratelimit-plugin/plugin"
)

// errUpdateGeneratedFiles:=errors.New("Error in generating files")
// Lint verifies code quality.
// func Lint() error {
// 	if err := sh.RunV("go", "generate", "./..."); err != nil {
// 		return err
// 	}

// 	if sh.Run("git", "diff", "--exit-code", "--", "'*.gen.go'") != nil {
// 		return errUpdateGeneratedFiles
// 	}

// 	if err := sh.RunV("go", "run", fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", golangCILintVer), "run"); err != nil {
// 		return err
// 	}

// 	if err := sh.RunV("go", "mod", "tidy"); err != nil {
// 		return err
// 	}

// 	if err := sh.RunV("go", "work", "sync"); err != nil {
// 		return err
// 	}

// 	if sh.Run("git", "diff", "--exit-code", "go.mod", "go.sum", "go.work", "go.work.sum") != nil {
// 		return errRunGoModTidy
// 	}

// 	return nil
// }

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
