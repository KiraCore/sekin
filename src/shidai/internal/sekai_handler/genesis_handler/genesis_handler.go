package genesishandler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
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
)

func GetVerifiedGenesisFile(ctx context.Context, ip, sekaidRPCPort, interxPort string) ([]byte, error) {
	log.Println("Getting verified genesis file")

	genesisSekaid, err := getSekaidGenesis(ctx, ip, sekaidRPCPort)
	if err != nil {
		return nil, err
	}
	genesisInterx, err := getInterxGenesis(ctx, ip, interxPort)
	if err != nil {
		return nil, err
	}

	if err := checkFileContentGenesisFiles(genesisInterx, genesisSekaid); err != nil {
		return nil, err
	}

	if err := checkGenSum(ctx, genesisSekaid, ip, interxPort); err != nil {
		return nil, err
	}

	return genesisSekaid, nil
}

// getSekaidGenesis retrieves the complete Sekaid Genesis data from a target Sekaid node
// by fetching the data in chunks using the Sekaid RPC API.
func getSekaidGenesis(ctx context.Context, ipAddress, sekaidRPCport string) ([]byte, error) {
	log.Println("getting sekaid genesis")
	var completeGenesis []byte
	var chunkCount int64
	client := &http.Client{}
	for {
		url := fmt.Sprintf("http://%s:%s/%s", ipAddress, sekaidRPCport, fmt.Sprintf("genesis_chunked?chunk=%d", chunkCount))

		chunkedGenesisResponseBody, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
		if err != nil {
			return nil, err
		}

		var response *ResponseChunkedGenesis
		err = json.Unmarshal([]byte(chunkedGenesisResponseBody), &response)
		if err != nil {
			return nil, err
		}

		totalValue, err := response.Result.Total.Int64()
		if err != nil {
			return nil, err
		}

		decodedData, err := base64.StdEncoding.DecodeString(response.Result.Data)
		if err != nil {
			return nil, err
		}

		completeGenesis = append(completeGenesis, decodedData...)

		chunkCount++
		if chunkCount >= totalValue {
			break
		}
	}

	return completeGenesis, nil
}

func getInterxGenesis(ctx context.Context, ipAddress, interxPort string) ([]byte, error) {
	log.Println("getting interx genesis")

	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, interxPort, "api/genesis")
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, err
	}

	return body, nil
}

func checkFileContentGenesisFiles(genesis1, genesis2 []byte) error {
	log.Println("checking file content")

	if string(genesis1) != string(genesis2) {
		return ErrFilesContentNotIdentical
	}

	return nil
}

// checkGenSum checks the integrity of a Genesis file using its SHA256 checksum.
func checkGenSum(ctx context.Context, genesis []byte, IpAddress, InterxPort string) error {
	log.Println("checking gen sum")

	genesisSum, err := getGenSum(ctx, IpAddress, InterxPort)
	if err != nil {
		return fmt.Errorf("can't get genesis check sum: %w", err)
	}

	genSumGenesisHash := sha256.Sum256(genesis)
	hashString := hex.EncodeToString(genSumGenesisHash[:])

	if genesisSum != hashString {
		return ErrSHA256ChecksumMismatch
	}

	return nil
}

// getGenSum retrieves the Genesis Sum from a target Interx server
// and returns it as a string after trimming the prefix "0x".
func getGenSum(ctx context.Context, ipAddress, interxPort string) (string, error) {
	log.Println("Getting gen sum")

	const genSumPrefix = "0x"
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, interxPort, "api/gensum")
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return "", err
	}

	var result *ResponseCheckSum

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	trimmedChecksum, err := trimPrefix(result.Checksum, genSumPrefix)
	if err != nil {
		return "", err
	}

	return trimmedChecksum, nil
}

type StringPrefixError struct {
	StringValue string
	Prefix      string
}

func (e *StringPrefixError) Error() string {
	return fmt.Sprintf("string '%s' does not have prefix '%s'", e.StringValue, e.Prefix)
}

// trimPrefix trims the specified prefix from the given string.
func trimPrefix(s, prefix string) (string, error) {
	if !strings.HasPrefix(s, prefix) {
		return "", &StringPrefixError{
			StringValue: s,
			Prefix:      prefix,
		}
	}

	return s[len(prefix):], nil
}
