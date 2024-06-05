package utils

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"regexp"

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
	var check bool = false
	for _, path := range infra {
		if !FileExists(path) {
			log.Warn("Infrastructure file not found", zap.String("path", path))
			check = false
		}
	}
	if check {
		log.Info("All infrastructure files are present")
	} else {
		log.Info("Not all infrastructure files are present")
	}

	return true && check
}
