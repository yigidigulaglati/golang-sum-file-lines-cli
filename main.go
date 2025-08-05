package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"
)

func validateDirs(dirs []string) error {
	var tempDir os.FileInfo
	var err error
	var absDir string
	for index, dir := range dirs {

		absDir, err = filepath.Abs(dir)
		if err != nil {
			fmt.Println(`Could not get absolute path for value:`, dir, `index:`, index, `Error:`, err.Error())
			return err
		}

		tempDir, err = os.Stat(absDir)
		if err != nil {
			fmt.Println(`Directory validation error at value:`, dir, `.`, `index:`, index, `Error:`, err.Error())
			if os.IsNotExist(err) {
				fmt.Println(`Directory does not exist.`)
			} else {
				return err
			}
		}

		if !tempDir.IsDir() {
			fmt.Println(`Directory validation error at value:`, dir, `.`, `index:`, index)
			fmt.Println(`Value`, dir, `is not a directory.`)
		}
	}

	return nil
}

func readLineCount(f *os.File) (int, error) {
	const bufferSize = 32 * 1024

	buf := make([]byte, bufferSize)
	var count int

	for {
		n, err := f.Read(buf)
		count += bytes.Count(buf[:n], []byte{'\n'})

		if err != nil {
			if err == io.EOF {
				return count, nil
			}

			return count, err
		}
	}
}

func handleRecDirs(dirs []string) int {
	var sum = 0
	var tempSum int
	var fileHandle *os.File
	for _, dir := range dirs {

		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if d.IsDir() && errors.Is(err, os.ErrPermission) {
					fmt.Println(`Permission error at`, path, `Error:`, err.Error())
					fmt.Println(`Skipping this directory.`)
					return filepath.SkipDir
				}
				fmt.Println(`Error at file or directory:`, path, `Error:`, err.Error())
				return nil
			}

			if d.IsDir() {
				return nil
			}
			
			
			absPath, err := filepath.Abs(path);
			if err != nil {
				fmt.Println(`Could not get the absolute path for this path:`, path, `Error:`, err);
				return nil;
			}
			
			fileHandle, err = os.OpenFile(absPath, os.O_RDONLY, 0400)
			
			if err != nil {
				fmt.Println(`Could not open file:`, absPath, `Error:`, err.Error())
				return nil
			}

			tempSum, err = readLineCount(fileHandle)
			
			sum += tempSum
			if err != nil {
				fmt.Println(`An error occurred reading file:`, absPath, `Error:`, err.Error())
				return nil
			}

			return nil
		})
	}

	return sum
}

func handleDirs(dirs []string) int {

	var names []os.DirEntry
	var err error
	var sum = 0
	var tempSum int
	var fileHandle *os.File
	var absPath string;
	for _, dir := range dirs {
		names, err = os.ReadDir(dir)
		
		if err != nil {
			continue
		}

		for _, dirEntry := range names {
			absPath, err = filepath.Abs(path.Join(dir, dirEntry.Name()));

			if err != nil {
				fmt.Println(`Could not get the absolute path for this file:`, dir, dirEntry.Name());
				continue;
			}

			if dirEntry.IsDir() {
				continue
			}
			
			fileHandle, err = os.OpenFile(absPath, os.O_RDONLY, 0400)

			if err != nil {
				fmt.Println(`Could not open the file:`, absPath, `Error:`, err.Error())
				continue
			}

			tempSum, err = readLineCount(fileHandle)
			sum += tempSum
			if err != nil {
				fmt.Println(`Error reading file:`, absPath, `Error:`, err.Error())
			}
		}
	}

	return sum
}

func handleFiles(files []string) int {

	var fileHandle *os.File
	var sum = 0
	var tempSum int
	var err error
	var absPath string;
	for _, file := range files {
		absPath, err = filepath.Abs(file);
		if err != nil {
			fmt.Println(`Could not get the absolute path for this file:`, file);
			continue;
		}

		fileHandle, err = os.Open(absPath)
		if err != nil {
			fmt.Println(`Could not open file:`, absPath, `Error:`, err.Error())
			continue
		}

		tempSum, err = readLineCount(fileHandle)
		sum += tempSum
		if err != nil {
			fmt.Println(`Error reading the file:`, absPath, `Error:`, err.Error())
		}
	}

	return sum
}

func main() {

	var TOTAL = 0

	var recDirs []string
	var dirs []string
	var files []string

	cmd := &cli.Command{

		Name:      `cl`,
		Version:   `1.0`,
		Usage:     `Sum the lines of files.`,
		UsageText: `Sum the lines of; files inside given directories (recursively and/or non-recursively) and/or given files.`,

		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        `rdir`,
				Aliases:     []string{`rd`},
				Value:       []string{},
				Destination: &recDirs,
				Usage:       `Use to recursively traverse the directory and add the line number of files to the total.`,
				DefaultText: `An empty slice of strings.`,
				Config: cli.StringConfig{
					TrimSpace: true,
				},
				Validator: func(dirs []string) error {
					err := validateDirs(dirs)
					if err != nil {
						return err
					}
					return nil
				},
			},

			&cli.StringSliceFlag{
				Name:        `dir`,
				Aliases:     []string{`d`},
				Value:       []string{},
				Destination: &dirs,
				Usage:       `Use to non-recursively traverse the directory and add the line number of files to the total.`,
				DefaultText: `An empty slice of strings.`,
				Config: cli.StringConfig{
					TrimSpace: true,
				},

				Validator: func(dirs []string) error {
					err := validateDirs(dirs)
					if err != nil {
						return err
					}
					return nil
				},
			},

			&cli.StringSliceFlag{
				Name:        `file`,
				Aliases:     []string{`f`},
				Value:       []string{},
				Destination: &files,
				Usage:       `Use to sum the line count of the file to the total.`,
				DefaultText: `An empty slice of strings.`,
				Config: cli.StringConfig{
					TrimSpace: true,
				},
				Validator: func(files []string) error {
					var absFilePath string
					var err error
					var fileStat os.FileInfo

					for index, file := range files {
						absFilePath, err = filepath.Abs(file)

						if err != nil {
							fmt.Println(`Could not get the absolute file path for the file:`, file, `index:`, index)
							fmt.Println(`Error:`, err.Error())
							return err
						}

						fileStat, err = os.Stat(absFilePath)
						if err != nil {
							fmt.Println(`File validation error at value:`, file, `index:`, index)
							if os.IsNotExist(err) {
								fmt.Println(`File does not exist.`, `Error:`, err.Error())
							}
							return err
						}

						if fileStat.IsDir() {
							fmt.Println(`Value:`, file, `is a directory, not a file.`, `index:`, index)
							return err
						}
					}
					return nil
				},
			},
		},

		Action: func(ctx context.Context, c *cli.Command) error {
			start := time.Now();
			TOTAL += handleDirs(dirs)
			TOTAL += handleRecDirs(recDirs)
			TOTAL += handleFiles(files)
			fmt.Println(`Time taken: `, time.Since(start));
			fmt.Println(`TOTAL LINES:`, TOTAL)

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}
