package utils

import (
	"evm-trackooor/shared"
	"fmt"
	"log/slog"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// formats n given its decimal places.
// truncate to 2 decimal places if formatted number > 0
// returns 0 if number == 0
func FormatDecimals(n *big.Int, decimals uint8) string {
	str := n.String()
	if decimals == 0 || n == nil || n.Cmp(big.NewInt(0)) == 0 {
		// dont format if no decimals or number is nil or number is 0
		return str
	}
	decs := int(decimals)
	l := len(str)
	if l > int(decimals) {
		pos := l - decs
		if decs > 2 {
			return str[:pos] + "." + str[pos:pos+2]
		}
		return str[:pos] + "." + str[pos:]
	}
	return strings.TrimRight("0."+strings.Repeat("0", decs-l)+str, "0")
}

// format into block scanner URL, like etherscan
// should return unformatted value if blockscanner data for it doesnt exist
// `typ` - type of entity, such as address, block, tx, token
func BlockscanFormat(typ string, value string) string {
	chain, ok := shared.Blockscanners[shared.ChainID.String()]
	if !ok {
		return value
	}
	chainMap := chain.(map[string]interface{})
	url, ok := chainMap[typ]
	if !ok {
		return value
	}
	urlStr := url.(string)
	return fmt.Sprintf(urlStr, value)
}

func CodeQuote(s string) string {
	if s == "" {
		return s
	}
	return "``" + s + "``"
}

func TrimFirstInstance(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func FormatBlockscanHyperlink(typ string, displayValue string, value string) string {
	url := BlockscanFormat(typ, value)
	// dont format if blockscan didn't format
	if url == value {
		return displayValue
	}
	return fmt.Sprintf("[`%v`](<%v>)", displayValue, url)
}

func FormatEventInfo(eventFields shared.EventFields, topics map[string]interface{}, data map[string]interface{}) (string, string) {
	// TODO handle structs for normal string output

	var output string
	var discordOutput string
	for _, eventField := range eventFields {
		var fieldValueFormatted string
		var fieldValueDiscordFormatted string
		fieldName, fieldType := eventField.FieldName, eventField.FieldType

		// find the value, either in topics or data
		fieldValue, ok := topics[fieldName]
		if !ok {
			fieldValue, ok = data[fieldName]
			if !ok {
				shared.Warnf(slog.Default(), "Couldn't find field value for field '%v'", fieldName)
				continue
			}
		}

		// convert field value to readable format

		if strings.HasPrefix(fieldType, "bytes") && !strings.HasSuffix(fieldType, "]") { // dynamic or static length bytes
			if fieldType == "bytes" && eventField.Indexed {
				// if indexed, value will be hash of the bytes
				fieldValueFormatted = fmt.Sprintf(
					"%v (keccak hash)",
					fieldValue,
				)
				fieldValueDiscordFormatted = fmt.Sprintf(
					"%v (keccak hash)",
					CodeQuote(fieldValue.(common.Hash).Hex()),
				)
			} else {
				// convert fixed byte array interface to []byte
				fieldValueLength := reflect.ValueOf(fieldValue).Len()
				decodedData := make([]byte, fieldValueLength)
				for i := 0; i < fieldValueLength; i++ {
					decodedData[i] = byte(reflect.ValueOf(fieldValue).Index(i).Uint())
				}
				fieldValueFormatted = fmt.Sprintf(
					"%v (UTF-8: %v)",
					"0x"+common.Bytes2Hex(decodedData),
					string(decodedData),
				)
				fieldValueDiscordFormatted = fmt.Sprintf(
					"`%v` (UTF-8: %v)",
					"0x"+common.Bytes2Hex(decodedData),
					CodeQuote(string(decodedData)),
				)
			}
		} else if fieldType == "string" {
			if eventField.Indexed {
				// if indexed, value will be hash of the string
				fieldValueFormatted = fmt.Sprintf(
					"%v (keccak hash)",
					fieldValue,
				)
				fieldValueDiscordFormatted = fmt.Sprintf(
					"%v (keccak hash)",
					CodeQuote(fieldValue.(common.Hash).Hex()),
				)
			} else {
				fieldValueFormatted = fmt.Sprintf("%v", fieldValue)
				fieldValueDiscordFormatted = fmt.Sprintf("%v", CodeQuote(fieldValue.(string)))
			}
		} else {
			fieldValueFormatted = fmt.Sprintf("%v", fieldValue)
			if fieldType == "address" {
				fieldValueDiscordFormatted = FormatBlockscanHyperlink(
					"address",
					fieldValue.(common.Address).Hex(),
					fieldValue.(common.Address).Hex(),
				)
			} else {
				fieldValueDiscordFormatted = fmt.Sprintf("`%v`", fieldValue)
			}
		}
		output += fmt.Sprintf(
			"%v: %v\n",
			fieldName,
			fieldValueFormatted,
		)
		discordOutput += fmt.Sprintf(
			"%v: %v\n",
			fieldName,
			fieldValueDiscordFormatted,
		)
	}
	return output, discordOutput
}

// makes addresses hex shorter
// e.g. 0xf05285270B723f389803cf6dCf15d253d087782b becomes 0xf0528...7782b
func ShortenAddress(addr common.Address) string {
	hexAddr := addr.Hex()
	return hexAddr[:2+5] + "..." + hexAddr[len(hexAddr)-5:]
}

// make hashes, e.g tx hashes shorter
func ShortenHash(hsh common.Hash) string {
	hshHex := hsh.Hex()
	return hshHex[:2+10] + "..." + hshHex[len(hshHex)-10:]
}
