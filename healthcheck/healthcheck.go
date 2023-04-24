package healthcheck

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
)

type Endpoint struct {
	Name    string            `yaml:"name"`
	Url     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Method  string            `yaml:"method,omitempty"`
	Body    string            `yaml:"body,omitempty"`
}

type pingStatus int

const (
	UNDEFINED pingStatus = iota
	UP
	DOWN
)

type pingResult struct {
	url    string
	status pingStatus
	err    error
}

type domainStat struct {
	upCount      int
	requestCount int
}

func (ds *domainStat) update(up bool) {
	if up {
		ds.upCount += 1
	}
	ds.requestCount += 1
}

func httpPing(ctx context.Context, client *http.Client, endpoint Endpoint) pingResult {

	method := http.MethodGet
	if len(endpoint.Method) > 0 {
		method = endpoint.Method
	}

	var bodyReader io.Reader
	if len(endpoint.Body) > 0 {
		bodyReader = strings.NewReader(endpoint.Body)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.Url, bodyReader)
	if err != nil {
		return pingResult{endpoint.Url, UNDEFINED, err}
	}

	for name, value := range endpoint.Headers {
		request.Header.Add(name, value)
	}

	response, err := client.Do(request)
	if err != nil {
		if urlError, ok := err.(*url.Error); ok && urlError.Timeout() {
			return pingResult{endpoint.Url, DOWN, nil}
		}
		return pingResult{endpoint.Url, UNDEFINED, err}
	}
	// TODO do I need to read the body to EOF and close?

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return pingResult{endpoint.Url, UP, nil}
	}
	return pingResult{endpoint.Url, DOWN, nil}
}

func PrintStats(domainList []string, domainStatsMap map[string]*domainStat) {

	for _, domain := range domainList {
		stats := domainStatsMap[domain]
		availability := int(math.Round(100 * float64(stats.upCount) / float64(stats.requestCount)))
		fmt.Printf("%v has %v%% availability percentage\n", domain, availability)
	}
}

func PeriodicHttpPing(ctx context.Context,
	client *http.Client,
	endpoints []Endpoint,
	pingInterval time.Duration,
	reportFunc func([]string, map[string]*domainStat),
) error {

	// create mapping from url to domain name
	domainNameMap := make(map[string]string)
	domainSet := mapset.NewSet[string]() // used to create domainList
	for _, endpoint := range endpoints {
		url, err := url.Parse(endpoint.Url)
		if err != nil {
			return err
		}
		domainNameMap[endpoint.Url] = url.Host
		domainSet.Add(url.Host)
	}

	// create list of unique domains for stable ordering
	domainList := domainSet.ToSlice()
	sort.Strings(domainList)

	// create mapping from domain name to domain stats
	domainStatsMap := make(map[string]*domainStat)
	for _, domain := range domainList {
		domainStatsMap[domain] = &domainStat{}
	}

	currentTime := time.Now()
	ticker := time.NewTicker(pingInterval)
	for {
		// create context with interval deadline
		dctx, dctxCancel := context.WithDeadline(ctx, currentTime.Add(pingInterval))

		// ping endpoints asynchronously
		pingResults := make(chan pingResult)
		for _, endpoint := range endpoints {
			go func(endpoint Endpoint) {
				pingResults <- httpPing(dctx, client, endpoint)
			}(endpoint)
		}
		for i := 0; i < len(endpoints); i++ {
			select {
			case result := <-pingResults:
				if result.err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", result.err)
					domainStatsMap[domainNameMap[result.url]].update(false)
				} else {
					domainStatsMap[domainNameMap[result.url]].update(result.status == UP)
				}
			case <-ctx.Done():
				dctxCancel()
				return nil
			}
		}
		dctxCancel()

		// report stats
		reportFunc(domainList, domainStatsMap)

		// wait for start of next interval or a canceled context
		select {
		case currentTime = <-ticker.C:
			continue
		case <-ctx.Done():
			return nil
		}
	}
}
