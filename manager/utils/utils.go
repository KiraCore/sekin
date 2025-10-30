package utils

import (
	"fmt"
	"github.com/saiset-co/sai-interx-manager/types"
	"strconv"
	"strings"
)

func ConvertRate(rateString string) string {
	rate, _ := strconv.ParseFloat(rateString, 64)
	rate = rate / 1000000000000000000.0
	rateString = fmt.Sprintf("%g", rate)
	if !strings.Contains(rateString, ".") {
		rateString = rateString + ".0"
	}
	return rateString
}

func ParseTimestamp(timestampStr string) int64 {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0
	}
	return timestamp
}

func TxСontainsAddress(tx types.TxResponse, address string) bool {
	for _, msg := range tx.Tx.Body.Messages {
		if msgData, ok := msg.(map[string]interface{}); ok {
			if sender, exists := msgData["sender"]; exists && sender == address {
				return true
			}
			if from, exists := msgData["from_address"]; exists && from == address {
				return true
			}
			if to, exists := msgData["to_address"]; exists && to == address {
				return true
			}
		}
	}
	return false
}

func MatchesTxType(tx types.TxResponse, typesList []string) bool {
	for _, msg := range tx.Tx.Body.Messages {
		if msgData, ok := msg.(map[string]interface{}); ok {
			if typeValue, exists := msgData["@type"]; exists {
				typeStr, _ := typeValue.(string)
				for _, txType := range typesList {
					if strings.Contains(typeStr, txType) {
						return true
					}
				}
			}
		}
	}
	return false
}

func DetermineDirection(tx types.TxResponse, address string) string {
	direction := "unknown"

	if len(tx.Logs) > 0 {
		for _, log := range tx.Logs {
			for _, event := range log.Events {
				if event.Type == "transfer" {
					for _, attr := range event.Attributes {
						if attr.Key == "sender" && attr.Value == address {
							direction = "out"
						} else if attr.Key == "recipient" && attr.Value == address {
							direction = "in"
						}
					}
				}
			}
		}
	} else {
		for _, msg := range tx.Tx.Body.Messages {
			if msgData, ok := msg.(map[string]interface{}); ok {
				msgType, _ := msgData["@type"].(string)

				if strings.Contains(msgType, "MsgSend") {
					if sender, exists := msgData["from_address"]; exists && sender == address {
						direction = "out"
					}
					if recipient, exists := msgData["to_address"]; exists && recipient == address {
						direction = "in"
					}
				}
			}
		}
	}

	return direction
}

func ParseTxMessages(tx types.TxResponse) []interface{} {
	var txResponses []interface{}

	for _, msg := range tx.Tx.Body.Messages {
		if msgData, ok := msg.(map[string]interface{}); ok {
			msgType, _ := msgData["@type"].(string)

			msgCopy := make(map[string]interface{})
			for k, v := range msgData {
				msgCopy[k] = v
			}
			msgCopy["type"] = msgType

			txResponses = append(txResponses, msgCopy)
		}
	}

	return txResponses
}
