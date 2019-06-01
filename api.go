package main

import (
	"crypto/tls"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/hashicorp/golang-lru"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"
)

const jsonAcceptType = "application/json"
const csvAcceptType = "text/csv"

const graylogOutputTimeFormat = "2006-01-02T15:04:05.000Z"
const graylogInputTimeFormat = "2006-01-02 15:04:05"

const relativeSearch = "search/universal/relative?range=%s"
const absoluteSearch = "search/universal/absolute?from=%s&to=%s"
const streamsInfo = "streams"

// Stores recent log messages. Graylog doesn't have any methods for preventing duplicates or overlaps, so we have to
// filter them out ourselves.
var msgCache, _ = lru.New(1024)

// Store the stream information so we don't have to pull it repeatedly.
var streamCache map[string]map[string]string

// Simple structure to hold a single log message.
type logMessage struct {
	id        string
	timestamp time.Time
	streams   []string
	fields    map[string]string
}

// Fetch all messages that match the settings in the options.
func fetchMessages(options *options) []logMessage {
	api, export := messageApiUri(options)
	var result []logMessage
	if export {
		callGraylog(options, api, csvAcceptType)
	} else {
		jsonBytes := callGraylog(options, api, jsonAcceptType)
		messages := getJsonArray(jsonBytes, "messages")
		_, _ = jsonparser.ArrayEach(messages, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			msg := getJsonSimpleMap(value, "message")
			tsStr := msg[timestampField]
			// Mon Jan 2 15:04:05 -0700 MST 2006

			ts, err := time.Parse(graylogOutputTimeFormat, tsStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid json timestamp: %s - %s\n", tsStr, err.Error())
			}
			if err == nil {
				streams := getJsonArrayOfStrings(value, "message", "streams")

				msgObj := logMessage{
					id:        string(msg["_id"]),
					timestamp: ts,
					streams:   streams,
					fields:    msg,
				}
				result = append(result, msgObj)
			}
		})
		sort.Slice(result, func(i, j int) bool {
			return result[i].timestamp.Before(result[j].timestamp)
		})

		if options.limit > 0 {
			var filteredMessages []logMessage
			for _, log := range result {
				if !msgCache.Contains(log.id) {
					filteredMessages = append(filteredMessages, log)
					msgCache.Add(log.id, true)
				}
			}
			result = filteredMessages
		}
	}

	return result
}

// Compute the API Uri to call. Determined by examing the command-line options.
func messageApiUri(options *options) (string, bool) {
	var uri string
	var export bool

	if options.startDate == nil || options.endDate == nil {
		uri = fmt.Sprintf(relativeSearch, strconv.Itoa(options.timeRange))
	} else {
		uri = fmt.Sprintf(absoluteSearch,
			url.QueryEscape((*options.startDate).Format(graylogInputTimeFormat)),
			url.QueryEscape((*options.endDate).Format(graylogInputTimeFormat)),
		)
		if len(options.fields) > 0 {
			export = true
			uri += "&fields=" + url.QueryEscape(options.fields)
		}
	}
	if options.limit > 0 && !export {
		uri += "&limit=" + strconv.Itoa(options.limit)
	}
	if len(options.query) > 0 {
		uri += "&query=" + url.QueryEscape(options.query)
	} else {
		uri += "&query=*"
	}

	if len(options.streamIds) > 0 {
		var searchTerm string
		for i, id := range options.streamIds {
			if i > 0 {
				searchTerm += " OR "
			}
			searchTerm += "streams:" + id
		}
		uri += "&filter=" + url.QueryEscape(searchTerm)
	}

	return uri, export
}

// Fetch the list of streams defined in Graylog.
func fetchStreams(options *options) map[string]map[string]string {
	if len(streamCache) > 0 {
		return streamCache
	}

	json := callGraylog(options, streamsInfo, jsonAcceptType)

	enabledStreams := make(map[string]map[string]string)

	slice := getJsonArray(json, "streams")
	if len(slice) > 0 {
		_, _ = jsonparser.ArrayEach(slice, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to read array entry: %s\n", err.Error())
			} else {
				disabled := getJsonBool(value, "disabled")
				id := getJsonString(value, "id")

				if !disabled {
					enabledStreams[id] = getJsonSimpleMap(value)
				}

			}
		})
	}

	streamCache = enabledStreams

	return enabledStreams
}

// Common entry-point for calls to Graylog.
func callGraylog(options *options, api string, acceptType string) []byte {
	cfg := options.serverConfig

	uri := cfg.Uri()
	username := cfg.Username()
	password := cfg.Password()
	ignoreCert := cfg.IgnoreCert()

	if acceptType == jsonAcceptType {
		return readBytes(uri+"/"+api, username, password, ignoreCert)
	}

	if acceptType == csvAcceptType {
		readCSV(uri+"/"+api, username, password, ignoreCert)
		return nil
	}
	return nil
}

// Return the raw bytes sent by Graylog.
func readBytes(uri string, username string, password string, ignoreCert bool) []byte {
	return fetch(uri, username, password, ignoreCert, jsonAcceptType)
}

// Process the results from Graylog as a CSV file.
func readCSV(uri string, username string, password string, ignoreCert bool) {
	fmt.Println("Exporting...")
	body := fetch(uri, username, password, ignoreCert, csvAcceptType)

	err := ioutil.WriteFile("export.csv", body, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to write to file 'export.csv': %s", err.Error())
	} else {
		cwd, _ := os.Getwd()
		fmt.Println("Contents exported to " + cwd + "/export.csv")
	}
}

// Low-level HTTP call to Graylog.
func fetch(uri string, username string, password string, ignoreCert bool, acceptType string) []byte {
	var client *http.Client
	if ignoreCert {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	} else {
		client = &http.Client{}
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Request is malformed: %s\n", err.Error())
		os.Exit(1)
	}
	if len(username) > 0 && len(password) > 0 {
		req.SetBasicAuth(username, password)
	}
	req.Header.Add("Accept", acceptType)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to Graylog: %s\n", err.Error())
		os.Exit(1)
	}
	//noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read content from Graylog: %s\n", err.Error())
		os.Exit(1)
	}

	return body
}
