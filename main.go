package main

import (
	"os"
	"time"
)

// Minimum delay between calls to Graylog
const MinDelay = 0.2
// Maximum delay between calls to Graylog
const MaxDelay = 30.0
// Back-off factor when increasing the delay.
const DelayIncreaseFactor = 2.0

// Adjust the delay between calls to Graylog so we don't hammer it when no messages have
// arrived for a while.
func adjustDelay(delay float64, messages []logMessage) float64 {
	if len(messages) == 0 {
		if delay < MaxDelay {
			delay *= DelayIncreaseFactor
			if delay > MaxDelay {
				delay = MaxDelay
			}
		}
	} else {
		delay = MinDelay
	}
	return delay
}

// Simple sleep function that uses a delay in seconds.
func sleep(delay float64) {
	delayInMilliseconds := int(delay * 1000.0)
	time.Sleep(time.Duration(delayInMilliseconds) * time.Millisecond)
}

func main() {
	options := ParseArgs()

	if options.listStreams {
		streams := fetchStreams(options)
		commandListStreams(streams)
		os.Exit(0)
	}

	if !options.tail {
		commandListMessages(options)
	} else {
		var delay = MinDelay

		//noinspection GoInfiniteFor
		for {
			messages := commandListMessages(options)

			sleep(delay)

			delay = adjustDelay(delay, messages)
		}
	}
}

