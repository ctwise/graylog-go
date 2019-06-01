package main

import (
	"sort"
	"strings"
)

// Converts a list of stream names into a list of stream ids.
func findStreamIds(options *options, streamNames string) []string {
	var results []string
	names := strings.Split(streamNames, ",")
	if len(names) > 0 {
		allStreams := fetchStreams(options)
		for _, name := range names {
			var id string
			lowerName := strings.ToLower(name)
			for _, v := range allStreams {
				streamName := strings.ToLower(v["title"])
				if strings.HasPrefix(streamName, lowerName) {
					id = v["id"]
				}
			}
			if len(id) > 0 {
				results = append(results, id)
			}
		}
	}

	return results
}

// Print out the list of streams defined in Graylog.
func commandListStreams(streams map[string]map[string]string) {
	var sts []map[string]string
	for _, v := range streams {
		sts = append(sts, v)
	}
	sort.Slice(sts, func(i, j int) bool {
		iTitle := strings.ToLower(string(sts[i]["title"]))
		jTitle := strings.ToLower(string(sts[j]["title"]))
		return iTitle < jTitle
	})

	for _, stream := range sts {
		description := stream["description"]
		title := stream["title"]
		if len(description) > 0 && title != description {
			printBoldText(title + " - " + description)
		} else {
			printBoldText(title)
		}
	}
}

// Print out the log messages that match the search criteria.
func commandListMessages(options *options) []logMessage {
	messages := fetchMessages(options)
	streams := fetchStreams(options)
	for _, msg := range messages {
		printMessage(options, streams, msg)
	}

	return messages
}
