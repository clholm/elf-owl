/*
   file copier & pr creator 🦉

    ,___,
    (o,o)
    /)_)
    ""
   tiny elf-owl
   watching over
   your files...
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/rand"
)

// executes a command and returns any error 🔧
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// finds all files in the given directory recursively 🔍
func findFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// convert to relative path
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}

// presents a fuzzy finder interface using fzf ✨
func selectFileWithFzf(files []string) (string, error) {
	// create fzf command
	cmd := exec.Command("fzf", "--height", "40%")

	// create pipes for stdin and stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// set stderr to the terminal
	cmd.Stderr = os.Stderr

	// start fzf 🚀
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start fzf: %v", err)
	}

	// write files to fzf
	go func() {
		defer stdin.Close()
		for _, file := range files {
			fmt.Fprintln(stdin, file)
		}
	}()

	// read selected file
	scanner := bufio.NewScanner(stdout)
	var selected string
	if scanner.Scan() {
		selected = scanner.Text()
	}

	// wait for fzf to exit
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", fmt.Errorf("file selection cancelled")
		}
		return "", fmt.Errorf("fzf failed: %v", err)
	}

	return strings.TrimSpace(selected), nil
}

// generates a branch name from filename and date 📅
func generateBranchName(filename string) string {
	date := time.Now().Format("06-01-02") // yy-mm-dd format
	// remove file extension and replace spaces/special chars with dashes
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, base)
	return fmt.Sprintf("%s-%s", base, date)
}

// returns a random happy emoji and bird emoji 🎲
func getRandomEmojis() (string, string) {
	happyEmojis := []string{"😊", "😃", "😄", "🙂", "😁", "😎"}
	birdEmojis := []string{"🐧", "🦉", "🦅", "🦆", "🦢", "🦜", "🦚", "🐤", "🦃", "🦅", "🦢", "🐦", "🕊️"}

	rand.Seed(uint64(time.Now().UnixNano()))
	return happyEmojis[rand.Intn(len(happyEmojis))], birdEmojis[rand.Intn(len(birdEmojis))]
}

// copies a file from src to dst 📋
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	return nil
}

// handles all git and github cli operations 🔄
func gitOperations(branchName, targetDir string) error {
	// change to target directory
	if err := os.Chdir(targetDir); err != nil {
		return fmt.Errorf("failed to change to target directory: %v", err)
	}

	// create and checkout new branch 🌿
	if err := runCommand("git", "checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch: %v", err)
	}

	// stage changes
	if err := runCommand("git", "add", "."); err != nil {
		return fmt.Errorf("failed to stage changes: %v", err)
	}

	// commit changes 📝
	if err := runCommand("git", "commit", "-m", fmt.Sprintf("Add %s", branchName)); err != nil {
		return fmt.Errorf("failed to commit changes: %v", err)
	}

	// push changes ⬆️
	if err := runCommand("git", "push", "--set-upstream", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push changes: %v", err)
	}

	// get two random emojis for the new pr
	happy, bird := getRandomEmojis()
	// create pr 🎯
	if err := runCommand("gh", "pr", "create",
		"--title", branchName,
		"--body", fmt.Sprintf("New finding! %s%s", happy, bird)); err != nil {
		return fmt.Errorf("failed to create pr: %v", err)
	}

	// open in browser 🌐
	if err := runCommand("gh", "browse"); err != nil {
		return fmt.Errorf("failed to open browser: %v", err)
	}

	return nil
}

func main() {
	// define flags 🚩
	searchDir := flag.String("search", "", "directory to search for files (required)")
	targetDir := flag.String("target", ".", "target directory (defaults to current directory)")
	branchName := flag.String("branch", "", "branch name (optional, will be generated from filename if not provided)")

	flag.Parse()

	// validate required flags
	if *searchDir == "" {
		fmt.Println("error: search directory is required")
		flag.Usage()
		os.Exit(1)
	}

	// validate that searchdir exists 🔍
	if _, err := os.Stat(*searchDir); os.IsNotExist(err) {
		fmt.Printf("error: search directory '%s' does not exist\n", *searchDir)
		os.Exit(1)
	}

	// convert paths to absolute ✨
	absSearchDir, err := filepath.Abs(*searchDir)
	if err != nil {
		fmt.Printf("error getting absolute path: %v\n", err)
		os.Exit(1)
	}
	absTargetDir, err := filepath.Abs(*targetDir)
	if err != nil {
		fmt.Printf("error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	// verify required commands exist 🛠️
	requiredCommands := []string{"fzf", "git", "gh"}
	for _, cmd := range requiredCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			fmt.Printf("error: required command '%s' not found in path\n", cmd)
			os.Exit(1)
		}
	}

	// find all files in search directory
	files, err := findFiles(absSearchDir)
	if err != nil {
		fmt.Printf("error finding files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("no files found in search directory '%s'\n", absSearchDir)
		os.Exit(1)
	}

	// select file using fzf ✨
	selectedFile, err := selectFileWithFzf(files)
	if err != nil {
		fmt.Printf("error selecting file: %v\n", err)
		os.Exit(1)
	}

	if selectedFile == "" {
		fmt.Println("no file selected")
		os.Exit(1)
	}

	// generate branch name if not provided 🌿
	finalBranchName := *branchName
	if finalBranchName == "" {
		finalBranchName = generateBranchName(selectedFile)
	}

	// source and destination paths 📂
	sourcePath := filepath.Join(absSearchDir, selectedFile)
	// use only the base filename for the destination
	destPath := filepath.Join(absTargetDir, filepath.Base(selectedFile))

	// copy the file 📋
	fmt.Printf("copying %s to %s...\n", sourcePath, destPath)
	if err := copyFile(sourcePath, destPath); err != nil {
		fmt.Printf("error copying file: %v\n", err)
		os.Exit(1)
	}

	// perform git operations 🔄
	fmt.Printf("performing git operations...\n")
	if err := gitOperations(finalBranchName, absTargetDir); err != nil {
		fmt.Printf("error in git operations: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("successfully completed all operations! 🎉")
}
