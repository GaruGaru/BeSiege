package main

import (
	"fmt"
	"sync"
	"os"
	"os/signal"
	"net/http"
	"time"
	"flag"
)

type result struct {
	success int
	fail    int
}

type siegeArgs struct {
	url         string
	concurrency int
	timeout     int
}

func parseArguments() siegeArgs {
	var concurrency = flag.Int("concurrency", 10, "Concurrency value (int)")
	var url = flag.String("url", "", "Selected url")
	var timeout = flag.Int("timeout", 1000, "Http connection timeout (ms)")
	flag.Parse()
	args := siegeArgs{*url, *concurrency, *timeout}
	validateArguments(args)
	return args
}

func main() {
	var siegeArgs siegeArgs = parseArguments()

	timeout := time.Duration(siegeArgs.timeout) * time.Millisecond

	var executionTime int64 = millis()
	var results_channel = make(chan result, siegeArgs.concurrency)
	var signal_channel = make(chan bool)
	var waitGroup sync.WaitGroup

	waitGroup.Add(siegeArgs.concurrency)

	fmt.Println("BeSiege, recruiting ", siegeArgs.concurrency, " concurrent warriors!")

	siege(siegeArgs.url, siegeArgs.concurrency, timeout, &waitGroup, signal_channel, results_channel)

	fmt.Println("BeSiege is running - Ctrl+C to exit")

	waitUserEscape()

	fmt.Println("Terminating ", siegeArgs.concurrency, " Go Routines...")

	close(signal_channel)
	waitGroup.Wait()

	executionTime = millis() - executionTime

	close(results_channel)
	summary := resultFromChannel(results_channel)

	printSummary(summary, executionTime)

	fmt.Println("BiSiege stopped, bye <3")
}

func validateArguments(args siegeArgs) {
	if len(args.url) == 0 {
		panic("Invalid url argument")
	}
	if args.concurrency <= 0 {
		panic("Invalid concurrency value")
	}
	if args.timeout <= 0 {
		panic("Invalid timeout value")
	}
}

func printSummary(summary result, executionTime int64) {
	fmt.Println("Requests: ", summary.success+summary.fail)
	fmt.Println("Success: ", summary.success)
	fmt.Println("Fail: ", summary.fail)
	fmt.Println("Throughtput: ", float64(summary.success+summary.fail)/float64(executionTime/1000), "req/sec")
}

func resultFromChannel(channel chan result) result {
	var totalSuccess int = 0
	var totalFail int = 0
	for r := range channel {
		totalSuccess += r.success
		totalFail += r.fail
	}
	return result{totalSuccess, totalFail}
}

func siege(url string, concurrency int, timeout time.Duration, waitGroup *sync.WaitGroup, signal_channel chan bool, results_channel chan result) {
	for i := 0; i < concurrency; i++ {
		go func(url string) {
			var success int = 0
			var fail int = 0

			httpClient := http.Client{Timeout: timeout}
			defer waitGroup.Done()
			defer func() { results_channel <- result{success, fail} }()

			for {
				select {
				case <-signal_channel:
					return
				default:
					result := request(httpClient, url)
					if result {
						success++
					} else {
						fail++
					}
				}
			}

		}(url)
	}
}

func millis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func request(client http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err.Error())
	} else if resp.StatusCode != 200 {
		fmt.Println(resp.StatusCode)
	}
	return err == nil && resp.StatusCode == 200
}

func waitUserEscape() {
	var end_waiter sync.WaitGroup
	end_waiter.Add(1)
	var signal_channel chan os.Signal
	signal_channel = make(chan os.Signal, 1)
	signal.Notify(signal_channel, os.Interrupt)
	go func() {
		<-signal_channel
		end_waiter.Done()
	}()
	end_waiter.Wait()
}
