package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	killSwitchTest := flag.Bool("kill-switch", false, "Run kill switch test")
	passthroughTest := flag.Bool("passthrough", false, "Run passthrough test")
	gracefulShutdownTest := flag.Bool("graceful-shutdown", false, "Run graceful shutdown test")
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("AggLayer Certificate Proxy Tests")
		fmt.Println("================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run tests/*.go -kill-switch        Run kill switch test")
		fmt.Println("  go run tests/*.go -passthrough        Run passthrough test")
		fmt.Println("  go run tests/*.go -graceful-shutdown  Run graceful shutdown test")
		return
	}

	if *killSwitchTest {
		runKillSwitchTest()
		return
	}

	if *passthroughTest {
		runPassthroughTest()
		return
	}

	if *gracefulShutdownTest {
		runGracefulShutdownTest()
		return
	}
}
