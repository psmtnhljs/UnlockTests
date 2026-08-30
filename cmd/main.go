package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/oneclickvirt/UnlockTests/executor"
	"github.com/oneclickvirt/UnlockTests/model"
	"github.com/oneclickvirt/UnlockTests/utils"
	. "github.com/oneclickvirt/defaultset"
)

type cliOptions struct {
	mode                                          int
	showVersion, help, showIP, useBar, cache, log bool
	jsonOutput, table                             bool
	iface, dnsServers, httpProxy, socksProxy      string
	language, selection, region, testNames        string
	concurrency                                   uint64
	timeout                                       time.Duration
	showIPSet, useBarSet, timeoutSet              bool
	selectionSet, regionSet, testNamesSet         bool
}

type cliStructuredOutput struct {
	SchemaVersion    string                          `json:"schema_version"`
	Results          []executor.StructuredResult     `json:"results"`
	ProviderMetadata []executor.ProviderMetadata     `json:"provider_metadata"`
	MetadataSource   executor.ProviderMetadataSource `json:"provider_metadata_source"`
	Error            string                          `json:"error,omitempty"`
}

func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{}
	fs := newFlagSet(&opts, io.Discard)
	if err := fs.Parse(normalizeRegionArgs(args)); err != nil {
		return opts, err
	}
	fs.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "s":
			opts.showIPSet = true
		case "b":
			opts.useBarSet = true
		case "timeout":
			opts.timeoutSet = true
		case "f":
			opts.selectionSet = true
		case "region":
			opts.regionSet = true
		case "test":
			opts.testNamesSet = true
		}
	})
	// The upstream CLI accepts both comma-separated region values and an
	// unquoted form such as `-region 0 11`. Go's flag package leaves the latter
	// values in Args, so fold them into the region expression before rejecting
	// positional arguments for every other mode.
	if fs.NArg() != 0 {
		if !opts.regionSet {
			return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
		}
		extra := strings.TrimSpace(strings.Join(fs.Args(), " "))
		if extra != "" {
			opts.region = strings.TrimSpace(strings.TrimSpace(opts.region) + " " + extra)
		}
	}
	opts.language = strings.ToLower(strings.TrimSpace(opts.language))
	if opts.help || opts.showVersion {
		return opts, nil
	}
	if opts.mode != 0 && opts.mode != 4 && opts.mode != 6 {
		return opts, fmt.Errorf("mode must be 0, 4, or 6")
	}
	if opts.language != "zh" && opts.language != "en" {
		return opts, fmt.Errorf("language must be zh or en")
	}
	if opts.selectionSet && opts.testNamesSet && strings.TrimSpace(opts.selection) != "" && strings.TrimSpace(opts.testNames) != "" {
		return opts, fmt.Errorf("-f and -test cannot be combined")
	}
	if opts.regionSet {
		if strings.TrimSpace(opts.region) == "" {
			return opts, fmt.Errorf("-region cannot be empty")
		}
		if opts.selectionSet && strings.TrimSpace(opts.selection) != "" {
			return opts, fmt.Errorf("-region and -f cannot be combined")
		}
		if opts.testNamesSet && strings.TrimSpace(opts.testNames) != "" {
			return opts, fmt.Errorf("-region and -test cannot be combined")
		}
		selection, err := executor.ParseRegionSelection(opts.region)
		if err != nil {
			return opts, err
		}
		opts.selection = selection
	}
	structuredMode := opts.jsonOutput || opts.table
	if opts.jsonOutput && opts.table {
		return opts, fmt.Errorf("-table cannot be combined with -json or -structured")
	}
	if structuredMode {
		if opts.showIPSet || opts.useBarSet {
			return opts, fmt.Errorf("-s and -b are not used with structured or table output")
		}
		if opts.timeoutSet && opts.timeout <= 0 {
			return opts, fmt.Errorf("structured/table timeout must be positive")
		}
	} else if opts.timeoutSet {
		return opts, fmt.Errorf("-timeout requires -json, -structured, or -table")
	}
	return opts, nil
}

// normalizeRegionArgs makes the upstream-friendly unquoted form
// "-region 0 11 -table" compatible with Go's flag parser. The standard
// parser stops at the first positional token, so collect only the contiguous
// values belonging to -region and leave every other flag untouched.
func normalizeRegionArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	result := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg != "-region" && arg != "--region" && !strings.HasPrefix(arg, "-region=") && !strings.HasPrefix(arg, "--region=") {
			result = append(result, arg)
			continue
		}

		value := ""
		if equal := strings.IndexByte(arg, '='); equal >= 0 {
			value = strings.TrimSpace(arg[equal+1:])
		}
		next := index + 1
		values := make([]string, 0, 4)
		if value != "" {
			values = append(values, value)
		}
		for next < len(args) && !isCLIFlagToken(args[next]) {
			values = append(values, args[next])
			next++
		}
		result = append(result, "-region", strings.Join(values, " "))
		index = next - 1
	}
	return result
}

func isCLIFlagToken(value string) bool {
	return len(value) > 1 && value[0] == '-'
}

func newFlagSet(opts *cliOptions, output io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("ut", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&opts.help, "h", false, "show help information")
	fs.IntVar(&opts.mode, "m", 0, "mode: 0 (both), 4 (only), or 6 (only); default is 0, example: -m 4")
	fs.BoolVar(&opts.showVersion, "v", false, "show version")
	fs.BoolVar(&opts.showIP, "s", true, "show IP address status; to disable, use: -s=false")
	fs.BoolVar(&opts.useBar, "b", true, "use progress bar; to disable, use: -b=false")
	fs.BoolVar(&opts.log, "log", false, "enable logging")
	fs.StringVar(&opts.selection, "f", "", "specify selection option in menu; example: -f 0")
	fs.StringVar(&opts.testNames, "test", "", "run specific providers by name or function, comma-separated; example: -test \"Coze,Poe\"")
	fs.StringVar(&opts.iface, "I", "", "bind IP address or network interface; example: -I 192.168.1.100 or -I eth0")
	fs.StringVar(&opts.dnsServers, "dns-servers", "", "specify DNS servers; example: -dns-servers \"1.1.1.1:53\"")
	fs.StringVar(&opts.httpProxy, "http-proxy", "", "specify HTTP proxy; example: -http-proxy \"http://username:password@127.0.0.1:1080\"")
	fs.StringVar(&opts.socksProxy, "socks-proxy", "", "specify SOCKS5 proxy; example: -socks-proxy \"socks5://username:password@127.0.0.1:1080\"")
	fs.Uint64Var(&opts.concurrency, "conc", 0, "max concurrent tests (0=structured default or legacy unlimited); example: -conc 50")
	fs.BoolVar(&opts.cache, "cache", false, "enable duplicate test result caching; example: -cache")
	fs.StringVar(&opts.language, "L", "zh", "language; specify 'en' for English or 'zh' for Chinese")
	fs.BoolVar(&opts.jsonOutput, "json", false, "print structured provider results as JSON")
	fs.BoolVar(&opts.jsonOutput, "structured", false, "print structured provider results as JSON")
	fs.StringVar(&opts.region, "region", "", "select regions by number or name (0-11, comma-separated; example: -region 0,11)")
	fs.BoolVar(&opts.table, "table", false, "print compact IPv4/IPv6 result tables")
	fs.DurationVar(&opts.timeout, "timeout", 0, "structured/table run timeout (for example 2m)")
	return fs
}

func main() {
	opts, err := parseCLI(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, sanitizeErrorText(err.Error()))
		os.Exit(2)
	}
	model.EnableLoger = opts.log
	if opts.help {
		fmt.Printf("Usage: %s [options]\n", os.Args[0])
		newFlagSet(&cliOptions{}, os.Stdout).PrintDefaults()
		return
	}
	if opts.showVersion {
		fmt.Println(model.UnlockTestsVersion)
		return
	}
	mode, showIP, useBar, cache := opts.mode, opts.showIP, opts.useBar, opts.cache
	Iface, DnsServers, httpProxy, socksProxy := opts.iface, opts.dnsServers, opts.httpProxy, opts.socksProxy
	language, flagString, testString, conc := opts.language, opts.selection, opts.testNames, opts.concurrency
	if err := executor.ValidateRunOptions(executor.RunOptions{Interface: Iface, DNSServers: DnsServers, HTTPProxy: httpProxy, SOCKSProxy: socksProxy}); err != nil {
		fmt.Fprintln(os.Stderr, sanitizeErrorText(err.Error()))
		os.Exit(2)
	}
	if opts.table {
		if conc > uint64(^uint(0)>>1) {
			fmt.Fprintln(os.Stderr, "concurrency exceeds the platform integer range")
			os.Exit(2)
		}
		ipVersion := "auto"
		if mode == 4 {
			ipVersion = "ipv4"
		} else if mode == 6 {
			ipVersion = "ipv6"
		}
		timeout := opts.timeout
		if timeout <= 0 {
			timeout = 2 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		runOptions := executor.RunOptions{
			Selection: flagString, IPVersion: ipVersion, Concurrency: int(conc), Interface: Iface,
			DNSServers: DnsServers, HTTPProxy: httpProxy, SOCKSProxy: socksProxy, UseCache: cache,
		}
		var results []executor.StructuredResult
		var runErr error
		if testString != "" {
			results, runErr = executor.RunNamedStructured(ctx, runOptions, testString)
		} else {
			results, runErr = executor.RunStructured(ctx, runOptions)
		}
		printStructuredTable(results)
		if runErr != nil {
			fmt.Fprintln(os.Stderr, sanitizeErrorText(runErr.Error()))
			os.Exit(1)
		}
		return
	}
	if opts.jsonOutput {
		output := cliStructuredOutput{SchemaVersion: "goecs.unlocktests/v1", Results: []executor.StructuredResult{}}
		if conc > uint64(^uint(0)>>1) {
			output.Error = "concurrency exceeds the platform integer range"
			encoded, _ := json.Marshal(output)
			fmt.Println(string(encoded))
			os.Exit(2)
		}
		ipVersion := "auto"
		if mode == 4 {
			ipVersion = "ipv4"
		} else if mode == 6 {
			ipVersion = "ipv6"
		}
		timeout := opts.timeout
		if timeout <= 0 {
			timeout = 2 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		metadata, metadataSource, metadataErr := executor.LoadProviderMetadata(ctx, nil)
		if metadataErr != nil {
			output.Error = sanitizeErrorText(metadataErr.Error())
			encoded, _ := json.Marshal(output)
			fmt.Println(string(encoded))
			os.Exit(1)
		}
		output.ProviderMetadata = metadata
		output.MetadataSource = metadataSource
		runOptions := executor.RunOptions{
			Selection: flagString, IPVersion: ipVersion, Concurrency: int(conc), Interface: Iface,
			DNSServers: DnsServers, HTTPProxy: httpProxy, SOCKSProxy: socksProxy, UseCache: cache,
		}
		var results []executor.StructuredResult
		var runErr error
		if testString != "" {
			results, runErr = executor.RunNamedStructured(ctx, runOptions, testString)
		} else {
			results, runErr = executor.RunStructured(ctx, runOptions)
		}
		output.Results = results
		if runErr != nil {
			output.Error = sanitizeErrorText(runErr.Error())
		}
		encoded, marshalErr := json.Marshal(output)
		if marshalErr != nil {
			fmt.Fprintln(os.Stderr, marshalErr)
			return
		}
		fmt.Println(string(encoded))
		if runErr != nil {
			os.Exit(1)
		}
		return
	}
	if Iface != "" {
		if err := executor.SetupInterface(Iface); err != nil {
			fmt.Fprintln(os.Stderr, sanitizeErrorText(err.Error()))
			os.Exit(1)
		}
	}
	if DnsServers != "" {
		executor.SetupDnsServers(DnsServers)
	}
	if httpProxy != "" {
		executor.SetupHttpProxy(httpProxy)
	}
	if socksProxy != "" {
		executor.SetupSocksProxy(socksProxy)
	}
	if conc > 0 {
		executor.SetupConcurrency(conc)
	}
	if cache {
		executor.EnableCache()
	}
	if mode == 4 {
		executor.IPV6 = false
	}
	if mode == 6 {
		executor.IPV4 = false
	}
	if language == "zh" {
		fmt.Fprintln(utils.ColorStdout, " 项目地址: "+Blue("https://github.com/oneclickvirt/UnlockTests"))
	} else {
		fmt.Fprintln(utils.ColorStdout, " Github Repo: "+Blue("https://github.com/oneclickvirt/UnlockTests"))
	}
	if testString == "" {
		readStatus := executor.ReadSelect(language, flagString)
		if !readStatus {
			return
		}
	}
	trackHit()
	if executor.IPV4 {
		executor.GetIpv4Info(showIP)
	}
	if executor.IPV6 {
		executor.GetIpv6Info(showIP)
	}
	if language == "zh" {
		fmt.Fprintln(utils.ColorStdout, " 测试时间: ", Yellow(time.Now().Format("2006-01-02 15:04:05")))
	} else {
		fmt.Fprintln(utils.ColorStdout, " Test time: ", Yellow(time.Now().Format("2006-01-02 15:04:05")))
	}
	if executor.IPV4 {
		if testString != "" {
			result, err := executor.RunNamedTests(utils.Ipv4HttpClient, "ipv4", language, useBar, testString)
			if err != nil {
				fmt.Fprintln(os.Stderr, sanitizeErrorText(err.Error()))
				os.Exit(1)
			}
			fmt.Fprint(utils.ColorStdout, indentLegacyOutput(result))
		} else {
			fmt.Fprint(utils.ColorStdout, indentLegacyOutput(executor.RunTests(utils.Ipv4HttpClient, "ipv4", language, useBar)))
		}
	}
	if executor.IPV6 {
		if testString != "" {
			result, err := executor.RunNamedTests(utils.Ipv6HttpClient, "ipv6", language, useBar, testString)
			if err != nil {
				fmt.Fprintln(os.Stderr, sanitizeErrorText(err.Error()))
				os.Exit(1)
			}
			fmt.Fprint(utils.ColorStdout, indentLegacyOutput(result))
		} else {
			fmt.Fprint(utils.ColorStdout, indentLegacyOutput(executor.RunTests(utils.Ipv6HttpClient, "ipv6", language, useBar)))
		}
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
	}
}

func trackHit() {
	go func() {
		client := utils.ReqDefault(utils.AutoHttpClient)
		client.SetTimeout(2 * time.Second)
		_, _ = client.R().Get("https://hits.spiritlhl.net/UnlockTests.svg?action=hit&title=Hits&title_bg=%23555555&count_bg=%230eecf8&edge_flat=false")
	}()
}
