// File: cmd/a8shub/main.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "a8shub",
	Short: "AgentsHub CLI tool",
}

var serverURL string

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8082", "AgentsHub server URL")

	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push [name:version] [file]",
	Short: "Push an agent file to the registry",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handlePush(serverURL, args[0], args[1])
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull [name:version]",
	Short: "Pull an agent file from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handlePull(serverURL, args[0])
	},
}

func handlePush(serverURL, ref, filePath string) error {
	// Parse name and version
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid reference format. Use name:version")
	}
	name, version := parts[0], parts[1]

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("agentfile", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}
	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// Add other form fields
	writer.WriteField("name", name)
	writer.WriteField("version", version)
	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/push", serverURL), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push failed: %s", string(bodyBytes))
	}

	fmt.Printf("Successfully pushed %s:%s\n", name, version)
	return nil
}

func handlePull(serverURL, ref string) error {
	// Parse name and version
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid reference format. Use name:version")
	}
	name, version := parts[0], parts[1]

	// Create request
	url := fmt.Sprintf("%s/v1/pull?name=%s&version=%s", serverURL, name, version)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed: %s", string(bodyBytes))
	}

	// Create output file
	outputPath := fmt.Sprintf("%s-%s.md", name, version)
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	// Copy content
	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write content: %v", err)
	}

	fmt.Printf("Successfully pulled %s:%s to %s\n", name, version, outputPath)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
