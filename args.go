package main

import (
	"./config"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/araddon/dateparse"
	"golang.org/x/sys/unix"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Limit used when no limit is provided by the user
const DefaultLimit = 300
// Range used when no range is provided by the user
const DefaultRange = "2h"
// Default location of the configuration path.
const DefaultConfigPath = "~/.graylog"

// Stores the command-line options and values.
type Options struct {
	listStreams  bool
	application  string
	query        string
	fields       string
	limit        int
	streamIds    []string
	tail         bool
	configPath   string
	timeRange    int
	startDate    *time.Time
	endDate      *time.Time
	json         bool
	serverConfig *config.ConfigFile
	noColor      bool
}

// Parse the command-line arguments.
// returns: *Options which contains both the parsed command-line arguments.
func ParseArgs() *Options {
	parser := argparse.NewParser("graylog", "Search and tail logs from Graylog.")

	var defaultConfigPath = expandPath(DefaultConfigPath)

	listStreams := parser.Flag("", "list-streams", &argparse.Options{Required: false, Help: "List Graylog streams and exit."})
	application := parser.String("a", "application", &argparse.Options{Required: false, Help: "Special case to search the 'application' message field, e.g., -a send-email is equivalent to -q 'application:send-email'. Merged with the -q query using 'AND' if the -q query is present."})
	query := parser.String("q", "query", &argparse.Options{Required: false, Help: "Query terms to search on (Elasticsearch syntax). Defaults to '*'."})
	fields := parser.String("e", "export", &argparse.Options{Required: false, Help: "Export specified fields as CSV into a file named 'export.csv'. Format is 'field1,field2,field3...'. Requires --start (and, optionally, --end) option."})
	limit := parser.Int("l", "limit", &argparse.Options{Required: false, Help: "The maximum number of messages to request from Graylog. Must be greater then 0", Default: DefaultLimit})
	streamNames := parser.String("s", "stream", &argparse.Options{Required: false, Help: "The name of the stream(s) to display messages from. Default: all streams."})
	tail := parser.Flag("t", "tail", &argparse.Options{Required: false, Help: "Whether to tail the output. Requires a relative search."})
	configPath := parser.String("c", "config", &argparse.Options{Required: false, Help: "Path to the config file", Default: defaultConfigPath})
	timeRange := parser.String("r", "range", &argparse.Options{Required: false, Help: "Time range to search backwards from the current moment. Examples: 30m, 2h, 4d", Default: DefaultRange})
	start := parser.String("", "start", &argparse.Options{Required: false, Help: "Starting time to search from. Allows variable formats, including '1:32pm' or '1/4/2019 12:30:00'."})
	end := parser.String("", "end", &argparse.Options{Required: false, Help: "Ending time to search from. Allows variable formats, including '6:45am' or '2019-01-04 12:30:00'. Defaults to now if --start is provided but no --end."})
	json := parser.Flag("j", "json", &argparse.Options{Required: false, Help: "Output messages in json format. Shows the modified log message, not the untouched message from Graylog. Useful in understanding the fields available when creating Format templates or for further processing."})
	noColor := parser.Flag("", "no-colors", &argparse.Options{Required: false, Help: "Don't use colors in output."})

	err := parser.Parse(os.Args)
	if err != nil {
		invalidArgs(parser, err, "")
	}
	if len(*fields) > 0 && len(*start) == 0 {
		invalidArgs(parser, nil, "The --export option requires the --start option")
	}

	startDate := strToDate(parser, *start, "The --start date can't be parsed", false)
	endDate := strToDate(parser, *end, "The --end date can't be parsed", true)

	if *limit <= 0 {
		var newLimit = DefaultLimit
		limit = &newLimit
	}

	if startDate != nil {
		var newTail = false
		tail = &newTail
	}

	var newQuery string
	if len(*application) > 0 {
		newQuery = "application:" + *application
		if len(*query) > 0 {
			newQuery += " AND " + *query
		}
		query = &newQuery
	}

	options := Options{
		listStreams: *listStreams,
		application: *application,
		query:       *query,
		fields:      *fields,
		limit:       *limit,
		tail:        *tail,
		configPath:  *configPath,
		timeRange:   timeRangeToSeconds(parser, *timeRange),
		startDate:   startDate,
		endDate:     endDate,
		json:        *json,
		noColor:     *noColor || isTty(),
	}

	// Read the configuration file
	cfg, err := config.New(options.configPath)
	if err != nil {
		invalidArgs(parser, err, "")
	}

	options.serverConfig = cfg

	// Convert the stream names into Graylog stream ids
	if len(*streamNames) > 0 {
		options.streamIds = findStreamIds(&options, *streamNames)
		if options.streamIds == nil {
			invalidArgs(parser, nil, "Invalid stream name(s)")
		}
	}

	return &options
}

// Convert a variable human-friendly date into a time.Time.
func strToDate(parser *argparse.Parser, dateStr string, errorStr string, defaultToNow bool) *time.Time {
	var dateTime time.Time
	var err error

	if len(dateStr) > 0 {
		// Check to see if the date is a time only
		matched, _ := regexp.MatchString("^[0-9]{1,2}:[0-9]{2}(:[0-9]{2})?([ ]*(am|pm|AM|PM)?)?$", dateStr)
		if matched {
			dateStr = time.Now().Format("2006-01-02") + " " + dateStr
		}
		dateTime, err = dateparse.ParseLocal(dateStr)
		if err != nil {
			invalidArgs(parser, err, errorStr)
		} else {
			if dateTime.Year() == 0 {
				dateTime = dateTime.AddDate(time.Now().Year(), 0, 0)
			}
		}
		if err != nil {
			return nil
		} else {
			return &dateTime
		}
	}
	if defaultToNow {
		dateTime = time.Now()
		return &dateTime
	} else {
		return nil
	}
}

// Converts a simple human-friendly time range into seconds, e.g., 2h for 2 hours, 3d2h30m for 3 days, 2 hours and
// 30 minutes.
func timeRangeToSeconds(parser *argparse.Parser, timeRange string) int {
	re := regexp.MustCompile("([0-9]*)([a-zA-Z]*)")
	parts := re.FindAllString(timeRange, -1)
	var accumulator int
	for _, part := range parts {
		if len(part) > 1 {
			unit := part[len(part)-1:]
			numberStr := part[:len(part)-1]
			num, err := strconv.Atoi(numberStr)
			if err != nil {
				invalidArgs(parser, err, "Time range can't be parsed")
			}
			switch strings.ToLower(unit) {
			case "s":
				accumulator += num
			case "m":
				accumulator += num * 60
			case "h":
				accumulator += num * 3600
			case "d":
				accumulator += num * 86400
			default:
				invalidArgs(parser, err, "Time range can't be parsed")
			}
		}
	}
	return accumulator
}

// Display the help message when a command-line argument is invalid.
func invalidArgs(parser *argparse.Parser, err error, msg string) {
	if len(msg) > 0 {
		if err != nil {
			fmt.Fprintf(os.Stderr,"%s: %s\n\n", msg, err.Error())
		} else {
			fmt.Fprintf(os.Stderr,"%s\n\n", msg)
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr,"%s\n\n", err.Error())
	}
	fmt.Fprintf(os.Stderr, parser.Usage(nil))
	os.Exit(1)
}

// Expand a leading tilde (~) in a file path into the user's home directory.
func expandPath(configPath string) string {
	var path = configPath
	if strings.HasPrefix(configPath, "~/") {
		usr, _ := user.Current()
		dir := usr.HomeDir

		// Use strings.HasPrefix so we don't match paths like
		// "/something/~/something/"
		path = filepath.Join(dir, path[2:])
	}
	return path
}

// Check to see whether we're outputting to a terminal or if we've been redirected to a file
func isTty() bool {
	_, err := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TIOCGETA)
	return err == nil
}
