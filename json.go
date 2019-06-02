package main

import (
	"fmt"
	"github.com/buger/jsonparser"
	"os"
	"strings"
)

// Retrieve a single value from the json buffer.
func getJSONValue(data []byte, keys ...string) (slice []byte, dataType jsonparser.ValueType, err error) {
	slice, dataType, _, err = jsonparser.Get(data, keys...)
	return slice, dataType, err
}

// Retrieve a single boolean value from the json buffer.
func getJSONBool(data []byte, keys ...string) bool {
	value, err := jsonparser.GetBoolean(data, keys...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to retrieve bool for keys: %v - %s\n", keys, string(err.Error()))
		return false
	}
	return value
}

// Retrieve a single string value from the json buffer.
func getJSONString(data []byte, keys ...string) string {
	value, err := jsonparser.GetString(data, keys...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to retrieve string for keys: %v - %s\n", keys, string(err.Error()))
		return ""
	}
	return Expand(value)
}

// Retrieve an array structure from the json buffer.
func getJSONArray(data []byte, keys ...string) []byte {
	slice, dataType, err := getJSONValue(data, keys...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to retrieve value for keys: %v - %s\n", keys, string(err.Error()))
	} else if dataType != jsonparser.Array {
		fmt.Fprintf(os.Stderr, "Key did not reference an array: %v\n", keys)
	} else {
		return slice
	}
	return []byte{}
}

// Retrieve a parsed array of strings from the json buffer.
func getJSONArrayOfStrings(data []byte, keys ...string) []string {
	arraySlice := getJSONArray(data, keys...)
	var stringList []string
	_, _ = jsonparser.ArrayEach(arraySlice, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if dataType == jsonparser.String || dataType == jsonparser.Number || dataType == jsonparser.Boolean {
			stringList = append(stringList, Expand(string(value)))
		}
	})
	return stringList
}

// Retrieve a parsed map of values from the json buffer. Numbers and booleans are converted to strings.
func getJSONSimpleMap(data []byte, keys ...string) map[string]string {
	result := make(map[string]string)
	_ = jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		if dataType == jsonparser.String || dataType == jsonparser.Number || dataType == jsonparser.Boolean {
			result[string(key)] = Expand(string(value))
		}
		return nil
	}, keys...)
	return result
}

// Expand escape strings. JSON strings from Graylog have embedded escape sequences that aren't getting expanded. We
// have to do it manually.
func Expand(value string) string {
	var result string
	result = strings.Replace(value, "\\n", "\n", -1)
	result = strings.Replace(result, "\\r", "\r", -1)
	result = strings.Replace(result, "\\t", "\t", -1)
	return result
}
