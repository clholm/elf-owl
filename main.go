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
)

// selectFileWithFzf presents a file selection interface using fzf
func selectFileWithFzf(files []string) (string, error) {
	// Create fzf command
	cmd := exec.Command("fzf", "--height", "40%")

	// Create pipes for stdin and stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Set stderr to the terminal
	cmd.Stderr = os.Stderr

	// Start fzf
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start fzf: %v", err)
	}

	// Write files to fzf
	go func() {
		defer stdin.Close()
		for _, file := range files {
			fmt.Fprintln(stdin, file)
		}
	}()

	// Read selected file
	scanner := bufio.NewScanner(stdout)
	var selected string
	if scanner.Scan() {
		selected = scanner.Text()
	}

	// Wait for fzf to exit
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", fmt.Errorf("file selection cancelled")
		}
		return "", fmt.Errorf("fzf failed: %v", err)
	}

	return strings.TrimSpace(selected), nil
}

// findFiles returns a list of all files in the given directory
func findFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Convert to relative path
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

// generates a branch name
func generateBranchName(filename string) string {
	date := time.Now().Format("06-01-02") // YY-MM-DD format
	// Remove file extension and replace spaces/special chars with dashes
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, base)
	return fmt.Sprintf("%s-%s", base, date)
}

// runCommand executes a command and returns any error
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
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

// gitOperations handles all git and GitHub CLI operations
func gitOperations(branchName, targetDir string) error {
	// Change to target directory
	if err := os.Chdir(targetDir); err != nil {
		return fmt.Errorf("failed to change to target directory: %v", err)
	}

	// Create and checkout new branch
	if err := runCommand("git", "checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch: %v", err)
	}

	// Add changes
	if err := runCommand("git", "add", "."); err != nil {
		return fmt.Errorf("failed to stage changes: %v", err)
	}

	// Commit changes
	if err := runCommand("git", "commit", "-m", fmt.Sprintf("Add %s", branchName)); err != nil {
		return fmt.Errorf("failed to commit changes: %v", err)
	}

	// Push changes
	if err := runCommand("git", "push", "--set-upstream", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push changes: %v", err)
	}

	// Create PR
	if err := runCommand("gh", "pr", "create",
		"--title", branchName,
		"--body", "New finding! 🙂🦉"); err != nil {
		return fmt.Errorf("failed to create PR: %v", err)
	}

	// Open in browser
	if err := runCommand("gh", "browse"); err != nil {
		return fmt.Errorf("failed to open browser: %v", err)
	}

	return nil
}

func main() {
	// Define flags
	searchDir := flag.String("search", "", "Directory to search for files (required)")
	targetDir := flag.String("target", ".", "Target directory (defaults to current directory)")
	branchName := flag.String("branch", "", "Branch name (optional, will be generated from filename if not provided)")

	flag.Parse()

	// Validate required flags
	if *searchDir == "" {
		fmt.Println("Error: search directory is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate that searchDir exists
	if _, err := os.Stat(*searchDir); os.IsNotExist(err) {
		fmt.Printf("Error: search directory '%s' does not exist\n", *searchDir)
		os.Exit(1)
	}

	// Convert paths to absolute
	absSearchDir, err := filepath.Abs(*searchDir)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}
	absTargetDir, err := filepath.Abs(*targetDir)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}

	// Verify required commands exist
	requiredCommands := []string{"fzf", "git", "gh"}
	for _, cmd := range requiredCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			fmt.Printf("Error: required command '%s' not found in PATH\n", cmd)
			os.Exit(1)
		}
	}

	// Find all files in search directory
	files, err := findFiles(absSearchDir)
	if err != nil {
		fmt.Printf("Error finding files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No files found in search directory '%s'\n", absSearchDir)
		os.Exit(1)
	}

	// Select file using fzf
	selectedFile, err := selectFileWithFzf(files)
	if err != nil {
		fmt.Printf("Error selecting file: %v\n", err)
		os.Exit(1)
	}

	if selectedFile == "" {
		fmt.Println("No file selected")
		os.Exit(1)
	}

	// Generate branch name if not provided
	finalBranchName := *branchName
	if finalBranchName == "" {
		finalBranchName = generateBranchName(selectedFile)
	}

	// Source and destination paths
	sourcePath := filepath.Join(absSearchDir, selectedFile)
	destPath := filepath.Join(absTargetDir, selectedFile)

	// Copy the file
	fmt.Printf("Copying %s to %s...\n", sourcePath, destPath)
	if err := copyFile(sourcePath, destPath); err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		os.Exit(1)
	}

	// Perform git operations
	fmt.Printf("Performing git operations...\n")
	if err := gitOperations(finalBranchName, absTargetDir); err != nil {
		fmt.Printf("Error in git operations: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully completed all operations!")
}
