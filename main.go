package main

import (
	"github.com/briandowns/spinner"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Minimum delay between calls to Graylog
const minDelay = 0.2

// Maximum delay between calls to Graylog
const maxDelay = 30.0

// Back-off factor when increasing the delay.
const delayIncreaseFactor = 2.0

// Adjust the delay between calls to Graylog so we don't hammer it when no messages have
// arrived for a while.
func adjustDelay(delay float64, messages []logMessage) float64 {
	if len(messages) == 0 {
		if delay < maxDelay {
			delay *= delayIncreaseFactor
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	} else {
		delay = minDelay
	}
	return delay
}

func delayForSeconds(delay float64) {
	delayInMilliseconds := int(delay * 1000.0)
	time.Sleep(time.Duration(delayInMilliseconds) * time.Millisecond)
}

func setupSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.UpdateCharSet(spinner.CharSets[21]) // box of dots
	s.Writer = os.Stderr
	s.HideCursor = true
	s.Color("red", "bold")
	return s
}

func makeSignalsChannel() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		// https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
		syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGQUIT, // Ctrl-\
		syscall.SIGKILL, // "always fatal", "SIGKILL and SIGSTOP may not be caught by a program"
		syscall.SIGHUP,  // "terminal is disconnected"
	)
	return c
}

func main() {
	opts := parseArgs()

	if opts.listStreams {
		streams := fetchStreams(opts)
		commandListStreams(streams)
		os.Exit(0)
	}

	if !opts.tail {
		messages, streams := commandListMessages(opts)
		printMessages(messages, opts, streams)
	} else {
		var delay = minDelay

		s := setupSpinner()
		s.Start()

		exitChan := makeSignalsChannel()

		// Handle exit signals - only needed when tailing
		go func() {
			for _ = range exitChan {
				s.Stop()
				os.Exit(0)
			}
		}()

		//noinspection GoInfiniteFor
		for {
			messages, streams := commandListMessages(opts)
			if len(messages) > 0 {
				s.Stop()
				printMessages(messages, opts, streams)
				s.Start()
			}

			delayForSeconds(delay)

			delay = adjustDelay(delay, messages)
		}
	}
}
