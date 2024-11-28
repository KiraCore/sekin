package genesishandler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"go.uber.org/zap"
)

type ResponseChunkedGenesis struct {
	Result struct {
		Chunk json.Number `json:"chunk"`
		Total json.Number `json:"total"`
		Data  string      `json:"data"`
	} `json:"result"`
}
type ResponseCheckSum struct {
	Checksum string `json:"checksum"`
}

var (
	ErrFilesContentNotIdentical = errors.New("files content are not identical")
	ErrSHA256ChecksumMismatch   = errors.New("sha256 checksum is not the same")

	log = logger.GetLogger()
)

func GetVerifiedGenesisFile(ctx context.Context, ip, sekaidRPCPort, interxPort string) ([]byte, error) {
	log.Info("Starting to get the verified genesis file", zap.String("IP", ip), zap.String("sekaidRPCPort", sekaidRPCPort), zap.String("interxPort", interxPort))

	// Get genesis file from Sekai daemon
	genesisSekaid, err := getSekaidGenesis(ctx, ip, sekaidRPCPort)
	if err != nil {
		log.Error("Failed to get genesis file from sekaid", zap.String("IP", ip), zap.String("Port", sekaidRPCPort), zap.Error(err))
		return nil, fmt.Errorf("failed to get sekaid genesis: %w", err)
	}
	log.Debug("Retrieved genesis file from sekaid", zap.ByteString("genesisSekaid", genesisSekaid))

	// Get genesis file from Interx daemon
	genesisInterx, err := GetInterxGenesis(ctx, ip, interxPort)
	if err != nil {
		log.Error("Failed to get genesis file from interx", zap.String("IP", ip), zap.String("Port", interxPort), zap.Error(err))
		return nil, fmt.Errorf("failed to get interx genesis: %w", err)
	}
	log.Debug("Retrieved genesis file from interx", zap.ByteString("genesisInterx", genesisInterx))

	// Check if both genesis files are the same
	if err := checkFileContentGenesisFiles(genesisInterx, genesisSekaid); err != nil {
		log.Error("Genesis files content mismatch", zap.Error(err))
		return nil, fmt.Errorf("genesis files content mismatch: %w", err)
	}
	log.Info("Genesis files content verified as matching")

	// Additional checksum verification
	if err := checkGenSum(ctx, genesisSekaid, ip, interxPort); err != nil {
		log.Error("Checksum verification failed", zap.Error(err))
		return nil, fmt.Errorf("checksum verification failed: %w", err)
	}
	log.Info("Checksum verification passed")

	log.Info("Genesis file verified successfully")
	return genesisSekaid, nil
}

// getSekaidGenesis retrieves the complete Sekaid Genesis data from a target Sekaid node
// by fetching the data in chunks using the Sekaid RPC API.
func getSekaidGenesis(ctx context.Context, ipAddress, sekaidRPCport string) ([]byte, error) {
	log.Info("Starting to get the sekaid genesis", zap.String("IP", ipAddress), zap.String("port", sekaidRPCport))

	var completeGenesis []byte
	var chunkCount int64
	client := &http.Client{}
	for {
		// Construct URL for the current chunk
		url := fmt.Sprintf("http://%s:%s/genesis_chunked?chunk=%d", ipAddress, sekaidRPCport, chunkCount)
		log.Debug("Requesting genesis chunk", zap.String("url", url))

		// Execute the HTTP request to get the genesis chunk
		chunkedGenesisResponseBody, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
		if err != nil {
			log.Error("Failed to get genesis chunk", zap.Int64("chunk", chunkCount), zap.Error(err))
			return nil, fmt.Errorf("failed to get genesis chunk %d: %w", chunkCount, err)
		}

		// Unmarshal JSON response
		var response *ResponseChunkedGenesis
		err = json.Unmarshal(chunkedGenesisResponseBody, &response)
		if err != nil {
			log.Error("Failed to unmarshal genesis chunk response", zap.Int64("chunk", chunkCount), zap.ByteString("response", chunkedGenesisResponseBody), zap.Error(err))
			return nil, fmt.Errorf("error unmarshaling response for chunk %d: %w", chunkCount, err)
		}

		// Decode the chunk data
		decodedData, err := base64.StdEncoding.DecodeString(response.Result.Data)
		if err != nil {
			log.Error("Failed to decode genesis data", zap.String("data", response.Result.Data), zap.Error(err))
			return nil, fmt.Errorf("error decoding genesis data for chunk %d: %w", chunkCount, err)
		}

		// Append decoded data to the complete genesis
		completeGenesis = append(completeGenesis, decodedData...)
		log.Debug("Appended genesis chunk", zap.Int64("chunk", chunkCount))

		// Increment the chunk count and check if we've received all chunks

		totalValue, err := response.Result.Total.Int64()
		if err != nil {
			return nil, err
		}
		chunkCount++
		if chunkCount >= totalValue {
			log.Info("All genesis chunks received", zap.Int64("total_chunks", totalValue))
			break
		}
	}

	log.Info("Complete sekaid genesis retrieved successfully")
	return completeGenesis, nil
}

func GetInterxGenesis(ctx context.Context, ipAddress, interxPort string) ([]byte, error) {
	log.Info("Starting to get the Interx genesis", zap.String("IP", ipAddress), zap.String("port", interxPort))
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	// Construct the URL for fetching the genesis data
	url := fmt.Sprintf("http://%s:%s/api/genesis", ipAddress, interxPort)
	log.Debug("Constructed URL for fetching Interx genesis", zap.String("url", url))

	// Create an HTTP client and perform the request
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		log.Error("Failed to get Interx genesis", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("failed to fetch Interx genesis from %s:%s: %w", ipAddress, interxPort, err)
	}

	log.Info("Interx genesis data retrieved successfully")
	return body, nil
}

func checkFileContentGenesisFiles(genesis1, genesis2 []byte) error {
	log.Info("Checking file content of two genesis files")
	if string(genesis1) != string(genesis2) {
		log.Error("Genesis files content does not match")
		return ErrFilesContentNotIdentical
	}
	log.Info("Genesis files content match confirmed")
	return nil
}

func checkGenSum(ctx context.Context, genesis []byte, ipAddress, interxPort string) error {
	log.Info("Checking Genesis checksum", zap.String("IP", ipAddress), zap.String("port", interxPort))
	genesisSum, err := getGenSum(ctx, ipAddress, interxPort)
	if err != nil {
		log.Error("Failed to retrieve genesis checksum from Interx", zap.Error(err))
		return fmt.Errorf("can't get genesis check sum: %w", err)
	}

	genSumGenesisHash := sha256.Sum256(genesis)
	hashString := hex.EncodeToString(genSumGenesisHash[:])
	if genesisSum != hashString {
		log.Error("Genesis checksum mismatch", zap.String("expected", genesisSum), zap.String("actual", hashString))
		return ErrSHA256ChecksumMismatch
	}

	log.Info("Genesis checksum verified successfully")
	return nil
}

func getGenSum(ctx context.Context, ipAddress, interxPort string) (string, error) {
	log.Info("Retrieving Genesis checksum", zap.String("IP", ipAddress), zap.String("port", interxPort))
	url := fmt.Sprintf("http://%s:%s/api/gensum", ipAddress, interxPort)
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		log.Error("Failed to fetch genesis sum", zap.Error(err))
		return "", err
	}

	var result ResponseCheckSum
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Error("Failed to unmarshal genesis sum response", zap.ByteString("response", body), zap.Error(err))
		return "", err
	}

	trimmedChecksum, err := trimPrefix(result.Checksum, "0x")
	if err != nil {
		log.Error("Failed to trim '0x' prefix from genesis sum", zap.String("checksum", result.Checksum), zap.Error(err))
		return "", err
	}

	log.Info("Genesis checksum retrieved and formatted", zap.String("checksum", trimmedChecksum))
	return trimmedChecksum, nil
}

type StringPrefixError struct {
	StringValue string
	Prefix      string
}

func (e *StringPrefixError) Error() string {
	return fmt.Sprintf("string '%s' does not have prefix '%s'", e.StringValue, e.Prefix)
}
func trimPrefix(s, prefix string) (string, error) {
	if !strings.HasPrefix(s, prefix) {
		log.Debug("String does not have the expected prefix", zap.String("string", s), zap.String("prefix", prefix))
		return "", &StringPrefixError{
			StringValue: s,
			Prefix:      prefix,
		}
	}
	return s[len(prefix):], nil
}
