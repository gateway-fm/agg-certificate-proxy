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
	sendTest := flag.Bool("send", false, "Send random certificate to local instance")
	metricsTest := flag.Bool("metrics", false, "Run metrics test")
	transparentProxyTest := flag.Bool("transparent-proxy", false, "Run transparent proxy test")
	integrityTest := flag.Bool("integrity", false, "Run integrity test")
	overrideTest := flag.Bool("overrides", false, "Run overrides test")
	suspiciousTest := flag.Bool("suspicious", false, "Run suspicious test")
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("AggLayer Certificate Proxy Tests")
		fmt.Println("================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run tests/*.go -kill-switch        Run kill switch test")
		fmt.Println("  go run tests/*.go -passthrough        Run passthrough test")
		fmt.Println("  go run tests/*.go -graceful-shutdown  Run graceful shutdown test")
		fmt.Println("  go run tests/*.go -send               Send random certificate to local instance")
		fmt.Println("  go run tests/*.go -metrics            Run metrics test")
		fmt.Println("  go run tests/*.go -transparent-proxy  Run transparent proxy test")
		fmt.Println("  go run tests/*.go -integrity          Run integrity test")
		fmt.Println("  go run tests/*.go -overrides          Run overrides test")
		fmt.Println("  go run tests/*.go -suspicious         Run suspicious test")
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

	if *sendTest {
		sendRandomCertificate()
		return
	}

	if *metricsTest {
		runMetricsIntegrationTest()
		return
	}

	if *transparentProxyTest {
		runTransparentProxyE2ETest()
		return
	}

	if *integrityTest {
		runDataIntegrityTest()
		return
	}

	if *overrideTest {
		runOverridesTest()
		return
	}

	if *suspiciousTest {
		runSuspiciousTest()
		return
	}
}
