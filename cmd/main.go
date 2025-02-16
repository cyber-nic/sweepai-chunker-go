package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"

	chunker "github.com/cyber-nic/chunker"
	goignore "github.com/cyber-nic/go-gitignore"
)

// main demonstrates chunking this file itself.
func main() {
	c := chunker.New(
		chunker.WithMaxChunkSize(1024),
		chunker.WithCoalesceThreshold(25),
	)

	globIgnorePatterns, err := goignore.CompileIgnoreFile(".astignore")
	if err != nil {
		log.Err(err).Msg("Error reading .astignore file")
		os.Exit(1)
	}

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		// Skip directories
		info, err = os.Stat(path)
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip files that match the ignore patterns
		if globIgnorePatterns.MatchesPath(path) {
			return nil
		}

		chunks, err := c.Harvest(path)
		if err != nil {
			log.Err(err).Msg("Error chunking file")
			return nil
		}

		fmt.Printf("\n\nParsed %s into %d chunks:\n", color.GreenString(path), len(chunks))
		for i, c := range chunks {
			fmt.Printf("\n%s\n%s\n", color.CyanString(fmt.Sprintf("--- (%d) lines %d-%d ---", i+1, c.Start, c.End)), c.Text)
		}

		return nil
	})

}
