package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

const debugEsc = "\033[94m"
const errorEsc = "\033[91m"
const infoEsc = "\033[92m"
const resetEsc = "\033[0;0m"
const warnEsc = "\033[93m"

const boldEsc = "\033[1m"

const debugLevel = "DEBUG"
const errorLevel = "ERROR"
const fatalLevel = "FATAL"
const infoLevel = "INFO"
const traceLevel = "TRACE"
const warnLevel = "WARN"

const longTimeFormat = "2006-01-02T15:04:05.000Z"

// Print a string in bold text.
func printBoldText(text string) {
	fmt.Println(boldEsc + text + resetEsc)
}

// Print a single log message
func printMessage(options *options, streamLookup map[string]map[string]string, msg logMessage) {
	adjustMessage(msg, streamLookup, options.noColor)

	var text string

	if options.json {
		buf, _ := json.Marshal(msg.fields)
		text = string(buf)
	} else {
		for _, f := range options.serverConfig.Formats() {
			text = tryFormat(msg, f.Name, f.Format)
			if len(text) > 0 {
				break
			}
		}
	}

	if len(text) > 0 {
		if strings.HasPrefix(text, "No Formats Defined>>") {
			fmt.Println("stop")
		}
		fmt.Println(text)
	} else {
		// Last case fallback in case none of the formats (including the default) match
		buf, _ := json.Marshal(msg.fields)
		fmt.Println(string(buf))
	}
}

// Try to apply a format template.
// returns: empty string if the format failed.
func tryFormat(msg logMessage, tmplName string, tmpl string) string {
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
	}
	var t = template.Must(template.New(tmplName).Option("missingkey=error").Funcs(funcMap).Parse(tmpl))
	var result bytes.Buffer
	err := t.Execute(&result, msg.fields)
	if err == nil {
		return result.String()
	} else {
		return ""
	}
}

// Convert a timestamp to a long time string.
func longTime(t time.Time) string {
	t = t.In(time.Local)
	return t.Format(longTimeFormat)
}

// "Cleanup" the log message and add helper fields.
func adjustMessage(msg logMessage, streamLookup map[string]map[string]string, isTty bool) {
	requestPage := msg.fields[requestPageField]
	if len(requestPage) > 1 && !strings.HasPrefix(requestPage, "/") {
		msg.fields[requestPageField] = "/" + requestPage
	}

	originalMessage := msg.fields[originalMessageField]
	if len(originalMessage) == 0 {
		originalMessage = msg.fields[fullMessageField]
		msg.fields[originalMessageField] = originalMessage
	}

	timestamp := msg.timestamp
	msg.fields[longTimestampField] = longTime(timestamp)

	classname := msg.fields[classnameField]
	if len(classname) > 0 {
		msg.fields[shortClassnameField] = createShortClassname(classname)
	}

	constructMessageText(msg, originalMessage)

	level := normalizeLevel(msg)

	if isTty {
		computeLogLevelColor(level, msg)
	} else {
		emptyLogLevelColor(msg)
	}

	if len(msg.streams) > 0 {
		var streamNames []string
		for _, streamId := range msg.streams {
			streamTitle := streamLookup[streamId]["title"]
			streamNames = append(streamNames, streamTitle)
		}
		streamDisplay := strings.Join(streamNames, " ")
		msg.fields[matchingStreamsField] = streamDisplay
	}
}

// Construct the "best" version of the log messages main text. This will look in multiple fields, attempt to
// append multi-line text (stacktraces) onto the message text, etc.
func constructMessageText(msg logMessage, originalMessage string) {
	const nestedException = "; nested exception "
	const newlineNnestedException = ";\nnested exception "

	messageText := msg.fields[messageField]
	if len(messageText) == 0 {
		messageText = originalMessage
	}
	if strings.Contains(messageText, nestedException) {
		messageText = strings.Replace(messageText, nestedException, newlineNnestedException, -1)
	}
	if len(originalMessage) > 0 && messageText != originalMessage {
		extraInfo := strings.Split(originalMessage, "\n")
		if len(extraInfo) == 2 {
			messageText = messageText + "\n" + extraInfo[1]
		}
		if len(extraInfo) > 2 {
			messageText = messageText + "\n" + strings.Join(extraInfo[1:len(extraInfo)-1], "\n")
		}
	}
	msg.fields[messageTextField] = messageText
}

// Normalize the "level" of the message.
func normalizeLevel(msg logMessage) string {
	level := msg.fields[logLevelField]
	if len(level) == 0 {
		level = msg.fields[levelField]
	}
	level = strings.ToUpper(level)
	if level == "WARNING" {
		level = warnLevel
	}
	msg.fields[logLevelField] = level
	return level
}

// Compute the color that should be used to display the log level in the message output.
func computeLogLevelColor(level string, msg logMessage) {
	var levelColor string
	switch level {
	case debugLevel, traceLevel:
		levelColor = debugEsc
	case infoLevel:
		levelColor = infoEsc
	case warnLevel:
		levelColor = warnEsc
	case errorLevel, fatalLevel:
		levelColor = errorEsc
	}
	if len(levelColor) > 0 {
		msg.fields[levelColorField] = levelColor
		msg.fields[resetField] = resetEsc
	} else {
		emptyLogLevelColor(msg)
	}
}

// Replace color strings with empty strings.
func emptyLogLevelColor(msg logMessage) {
	msg.fields[levelColorField] = ""
	msg.fields[resetField] = ""
}

// Create a shortened version of the Java classname.
func createShortClassname(classname string) string {
	parts := strings.Split(classname, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return classname
}
