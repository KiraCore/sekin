package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/tyler-smith/go-bip39"
	"go.uber.org/zap"
)

var log = logger.GetLogger()

func ContainsValue(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

// ValidateIP checks if the given string is a valid IPv4 or IPv6 address.
// It returns true if the IP is valid, otherwise returns false.
func ValidateIP(ip string) bool {
	isValid := net.ParseIP(ip) != nil
	return isValid
}

// ValidatePort checks if the given port number is within the valid range of 1 to 65535.
// It returns true if the port is valid, otherwise returns false.
func ValidatePort(port int) bool {
	isValid := port > 0 && port <= 65535
	return isValid
}

// ValidateMnemonic checks if the given mnemonic is valid according to the BIP-0039 standard.
// It returns true if the mnemonic is valid, otherwise returns false.
func ValidateMnemonic(mnemonic string) bool {
	isValid := bip39.IsMnemonicValid(mnemonic)
	return isValid
}

// IsPublicIP checks if the given IP address is a public IP address.
func IsPublicIP(ip net.IP) bool {
	privateIPBlocks := []*regexp.Regexp{
		regexp.MustCompile(`^10\..*`),
		regexp.MustCompile(`^172\.(1[6-9]|2[0-9]|3[0-1])\..*`),
		regexp.MustCompile(`^192\.168\..*`),
	}
	ipStr := ip.String()
	for _, block := range privateIPBlocks {
		if block.MatchString(ipStr) {
			return false
		}
	}
	return true
}

// GetPublicIP retrieves the public IP address of the system.
// Returns an error if more than one public IP address is found.
func GetPublicIP() (string, error) {
	var publicIPs []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Error("failed toget interface addresses")
		return "", fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip != nil && !ip.IsLoopback() && ip.To4() != nil && IsPublicIP(ip) {
			publicIPs = append(publicIPs, ip.String())
		}
	}

	if len(publicIPs) == 0 {
		log.Warn("no public IP addresses found")
		return "", fmt.Errorf("no public IP addresses found")
	}
	if len(publicIPs) > 1 {
		log.Warn("multiple public IP addresses found")
		return "", fmt.Errorf("multiple public IP addresses found")
	}

	return publicIPs[0], nil
}

func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	isFile := !info.IsDir()
	return isFile
}

// DeleteFile removes a file specified by the file path.
func DeleteFile(filePath string) error {
	log.Info("attempting to delete file", zap.String("path", filePath))
	err := os.Remove(filePath)
	if err != nil {
		log.Error("failed to delete fiel", zap.String("path", filePath))
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	log.Info("succefully deleted the file", zap.String("path", filePath))
	return nil
}

func CreateDir(path string, perm os.FileMode) error {
	log.Info("creating directory", zap.String("path", path))
	err := os.MkdirAll(path, perm)
	if err != nil {
		log.Error("failed to create directory", zap.String("path", path), zap.Error(err))
		return fmt.Errorf("failed to create a directory %s: %w", path, err)
	}

	log.Info("succefully created directory", zap.String("path", path))
	return nil
}

// LoadConfig loads config.toml to Config structure
func LoadConfig(filePath string, config types.Config) error {
	if _, err := toml.DecodeFile(filePath, config); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	return nil
}

// LoadAppConfig loads app.toml to ConfigApp structure
func LoadAppConfig(filePath string, config types.AppConfig) error {
	if _, err := toml.DecodeFile(filePath, config); err != nil {
		return fmt.Errorf("failed to load app config: %w", err)
	}
	return nil
}

// SaveConfig saves config.toml to given path
func SaveConfig(filePath string, config types.Config) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	return nil
}

// SaveAppConfig saves app.toml to given path
func SaveAppConfig(filePath string, config types.AppConfig) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create app config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode app config: %w", err)
	}
	return nil
}

// SetField sets a value on a given struct field and returns a description of the change.
func SetField(obj interface{}, fieldName string, newValue interface{}) (string, error) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return "", fmt.Errorf("expected a pointer to a struct")
	}

	v = v.Elem() // Dereference the pointer to get the struct

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return "", fmt.Errorf("no such field: %s in obj", fieldName)
	}
	if !field.CanSet() {
		return "", fmt.Errorf("cannot set field %s", fieldName)
	}

	fieldType := field.Type()
	newVal := reflect.ValueOf(newValue)
	if newVal.Type() != fieldType {
		return "", fmt.Errorf("provided value type %s doesn't match obj field type %s", newVal.Type(), fieldType)
	}

	oldValue := fmt.Sprintf("%v", field.Interface())
	field.Set(newVal)
	changeDescription := fmt.Sprintf("Changed %s from %s to %v", fieldName, oldValue, newValue)

	return changeDescription, nil
}

func CheckInfra(infra types.InfraFiles) bool {
	allFilesPresent := true // Assume all files are present initially

	for _, path := range infra {
		if !FileExists(path) {
			log.Warn("Infrastructure file not found", zap.String("path", path))
			allFilesPresent = false // Set to false if any file is missing
		}
	}

	if allFilesPresent {
		log.Info("All infrastructure files are present")
	} else {
		log.Info("Not all infrastructure files are present")
	}

	return allFilesPresent // Return true if all files are present, false otherwise
}
func GenerateRandomString(n int) string {
	const lettersAndDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		log.Error("Failed to generate random string", zap.Error(err))
	}
	for i := 0; i < n; i++ {
		b[i] = lettersAndDigits[b[i]%byte(len(lettersAndDigits))]
	}
	return string(b)
}

// GetChainID fetches and parses the chain_id from a predefined URL
func GetChainID(url string) (string, error) {
	// Define the type locally within the function
	type DashboardData struct {
		ChainID string `json:"chain_id"`
	}

	log.Debug("Sending GET request to URL", zap.String("url", url))

	// Send a GET request to the URL
	resp, err := http.Get(url)
	if err != nil {
		log.Error("Failed to fetch data", zap.Error(err))
		return "", fmt.Errorf("error fetching data: %w", err)
	}
	defer resp.Body.Close()
	log.Debug("Received response from server")

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", zap.Error(err))
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	log.Debug("Response body read successfully", zap.ByteString("body", body))

	// Parse the JSON data
	var data DashboardData
	if err := json.Unmarshal(body, &data); err != nil {
		log.Error("Failed to parse JSON", zap.Error(err))
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}
	log.Debug("JSON parsed successfully", zap.String("chain_id", data.ChainID))

	// Check if the chain_id is empty, which could indicate missing data
	if data.ChainID == "" {
		log.Warn("Chain ID is missing in the response")
		return "", fmt.Errorf("chain_id is missing in the response")
	}

	log.Info("Successfully retrieved chain ID", zap.String("chain_id", data.ChainID))
	// Return the chain_id
	return data.ChainID, nil
}

func SafeCopy(src, dst string) error {
	log.Debug("Trying to copy <%v> to <%v>", zap.String("source:", src), zap.String("destination:", dst))
	check := FileExists(src)
	if !check {
		return fmt.Errorf("source file does not exist")
	}

	info, err := os.Stat(dst)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("destination path <%v> is a folder, cannot overwrite", dst)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to access destination path <%v>: %w", dst, err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	if err := syscall.Flock(int(srcFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("file is locked: %w", err)
		} else {
			return fmt.Errorf("failed to lock source file: %w", err)
		}
	}

	defer syscall.Flock(int(srcFile.Fd()), syscall.LOCK_UN)

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if err := syscall.Flock(int(dstFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("destination file is locked: %w", err)
		} else {
			return fmt.Errorf("failed to lock destination file: %w", err)
		}
	}
	defer syscall.Flock(int(dstFile.Fd()), syscall.LOCK_UN)

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	log.Debug("File copied", zap.String("source:", src), zap.String("destination:", dst))
	return nil
}

func FilesAreEqual(file1, file2 string) (bool, error) {
	f1, err := os.Open(file1)
	if err != nil {
		return false, fmt.Errorf("failed to open file1: %w", err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		return false, fmt.Errorf("failed to open file2: %w", err)
	}
	defer f2.Close()

	buf1 := make([]byte, 1024)
	buf2 := make([]byte, 1024)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		if err1 != nil && err1 != io.EOF {
			return false, fmt.Errorf("failed to read from file1: %w", err1)
		}
		if err2 != nil && err2 != io.EOF {
			return false, fmt.Errorf("failed to read from file2: %w", err2)
		}

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			break
		}
	}

	return true, nil
}

func FilesAreEqualMD5(file1, file2 string) (bool, error) {
	hash1, err := HashFileMD5(file1)
	if err != nil {
		return false, fmt.Errorf("failed to hash file1: %w", err)
	}

	hash2, err := HashFileMD5(file2)
	if err != nil {
		return false, fmt.Errorf("failed to hash file2: %w", err)
	}

	return hash1 == hash2, nil
}

func HashFileMD5(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
