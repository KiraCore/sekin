package utils

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	isFile := !info.IsDir()
	return isFile
}

func DeleteFile(filePath string) error {
	log.Printf("attempting to delete file <%v>", filePath)
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("failed to delete file <%v>", filePath)
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	log.Printf("successfully deleted the file <%v>", filePath)
	return nil
}

func CopyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get the source file's mode (permissions)
	sourceFileInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	sourceMode := sourceFileInfo.Mode()

	// Create the destination file with the same permissions as the source file
	destinationFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceMode)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

func UpdateComposeYmlField(compose map[string]interface{}, serviceName, fieldName, newValue string) {
	if services, ok := compose["services"].(map[interface{}]interface{}); ok {
		if service, ok := services[serviceName].(map[interface{}]interface{}); ok {
			service[fieldName] = newValue
		}
	}
}

// RenameFile renames a file from oldName to newName
func RenameFile(oldName, newName string) error {
	// Use os.Rename to rename the file
	err := os.Rename(oldName, newName)
	if err != nil {
		return fmt.Errorf("error renaming file: %v", err)
	}
	return nil
}

func DoHttpQuery(ctx context.Context, client *http.Client, url, method string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {

		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// log.Debug("Response body read successfully", zap.ByteString("response", body))

	return body, nil
}
