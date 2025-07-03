package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SubCombination struct {
	wordlist   string
	timeout    time.Duration
	client     *http.Client
	resolver   *net.Resolver
	maxWorkers int
}

type ProgressTracker struct {
	total     int64
	completed int64
	startTime time.Time
	ticker    *time.Ticker
	done      chan bool
}

func NewProgressTracker(total int64) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		completed: 0,
		startTime: time.Now(),
		ticker:    time.NewTicker(500 * time.Millisecond), // Atualiza a cada 500ms
		done:      make(chan bool),
	}
}

func (pt *ProgressTracker) Start() {
	go func() {
		for {
			select {
			case <-pt.ticker.C:
				pt.display()
			case <-pt.done:
				pt.ticker.Stop()
				pt.display() 
				fmt.Println()
				return
			}
		}
	}()
}

func (pt *ProgressTracker) Increment() {
	atomic.AddInt64(&pt.completed, 1)
}

func (pt *ProgressTracker) Stop() {
	close(pt.done)
}

func (pt *ProgressTracker) display() {
	completed := atomic.LoadInt64(&pt.completed)
	percentage := float64(completed) / float64(pt.total) * 100
	elapsed := time.Since(pt.startTime)

	var eta time.Duration
	if completed > 0 {
		remainingItems := pt.total - completed
		timePerItem := elapsed / time.Duration(completed)
		eta = timePerItem * time.Duration(remainingItems)
	}

	rate := float64(completed) / elapsed.Seconds()

	barWidth := 40
	filledWidth := int(float64(barWidth) * percentage / 100)
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("▒", barWidth-filledWidth)
	
	fmt.Printf("\r[%s] %.1f%% (%d/%d) | %.1f/s | ETA: %v | Elapsed: %v",
		bar, percentage, completed, pt.total, rate, eta.Round(time.Second), elapsed.Round(time.Second))
}

func NewSubCombination() *SubCombination {
	// Cliente HTTP 
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 30,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 3 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, 
		},
		DisableKeepAlives: false,
	}

	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Resolver DNS 
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 2 * time.Second,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	return &SubCombination{
		wordlist:   "",
		timeout:    3 * time.Second,
		client:     client,
		resolver:   resolver,
		maxWorkers: 300,
	}
}

func (sc *SubCombination) wordList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" { 
			words = append(words, word)
		}
	}
	return words, scanner.Err()
}

func (sc *SubCombination) generateSubdomains(sub string, words []string, debug bool) []string {
	parts := strings.Split(sub, ".")
	lenSubdomain := len(parts)
	estimatedSize := (lenSubdomain - 2) * len(words)
	if estimatedSize <= 0 {
		estimatedSize = len(words)
	}
	subdomains := make([]string, 0, estimatedSize)

	for i := 0; i < lenSubdomain-2; i++ {
		for _, word := range words {
			var builder strings.Builder
			builder.Grow(len(sub) + len(word) + 1)
			for j := 0; j <= i; j++ {
				builder.WriteString(parts[j])
				builder.WriteString(".")
			}
			builder.WriteString(word)
			builder.WriteString(".")
			for j := i + 1; j < len(parts); j++ {
				builder.WriteString(parts[j])
				if j < len(parts)-1 {
					builder.WriteString(".")
				}
			}
			domain := builder.String()
			if debug {
				fmt.Println("[DEBUG] Gerado:", domain)
			}
			subdomains = append(subdomains, domain)
		}
	}
	return subdomains
}

// Verifica DNS 
func (sc *SubCombination) checkDNS(domain string, debug bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := sc.resolver.LookupHost(ctx, domain)
	if err != nil && debug {
		fmt.Printf("[DEBUG] DNS failed for %s\n", domain)
	}
	return err == nil
}

func (sc *SubCombination) checkSubdomainAlive(domain string, progress *ProgressTracker, debug bool) *string {
	defer progress.Increment()

	// Primeiro verifica DNS
	if !sc.checkDNS(domain, debug) {
		return nil
	}

	schemes := []string{"https://", "http://"}
	for _, scheme := range schemes {
		url := scheme + domain + "/"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if debug {
				fmt.Printf("[DEBUG] Error creating GET request for %s: %v\n", url, err)
			}
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "*/*")

		ctx, cancel := context.WithTimeout(context.Background(), sc.timeout)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := sc.client.Do(req)
		if err != nil {
			if debug {
				fmt.Printf("[DEBUG] Error on GET request for %s: %v\n", url, err)
			}
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("\r%s\n", strings.Repeat(" ", 120))
		fmt.Printf("[+] ONLINE: %s (%d)\n", domain, resp.StatusCode)
		return &domain
	}

	return nil
}

func (sc *SubCombination) bruteDomains(subList []string, debug bool) []string {
	words, err := sc.wordList(sc.wordlist)
	if err != nil {
		fmt.Printf("Error reading wordlist: %v\n", err)
		return nil
	}

	fmt.Printf("[*] Loaded %d words from wordlist\n", len(words))
	fmt.Println("[*] Generating subdomains...")

	subdomainChan := make(chan []string, len(subList))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limite para geração

	for _, sub := range subList {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			results := sc.generateSubdomains(s, words, debug)
			subdomainChan <- results
		}(sub)
	}

	// Goroutine para fechar o canal
	go func() {
		wg.Wait()
		close(subdomainChan)
	}()

	// Coleta todos os subdomínios
	var allSubdomains []string
	for results := range subdomainChan {
		allSubdomains = append(allSubdomains, results...)
	}

	if debug {
		fmt.Printf("\n[*] Generated %d subdomains (debug mode, exiting)\n", len(allSubdomains))
		return allSubdomains
	}

	fmt.Printf("\n[*] Checking %d subdomains...\n", len(allSubdomains))

	progress := NewProgressTracker(int64(len(allSubdomains)))
	progress.Start()

	resultChan := make(chan string, sc.maxWorkers)
	checkSemaphore := make(chan struct{}, sc.maxWorkers)

	wg = sync.WaitGroup{}
	for _, domain := range allSubdomains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			checkSemaphore <- struct{}{}
			defer func() { <-checkSemaphore }()

			if result := sc.checkSubdomainAlive(d, progress, debug); result != nil {
				resultChan <- *result
			}
		}(domain)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		progress.Stop()
	}()

	var aliveSubdomains []string
	for domain := range resultChan {
		aliveSubdomains = append(aliveSubdomains, domain)
	}

	return aliveSubdomains
}

func (sc *SubCombination) writeOut(domains []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, domain := range domains {
		if _, err := writer.WriteString(domain + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	domain := flag.String("d", "", "Domain to process")
	wordlistFile := flag.String("w", "", "Wordlist file for subdomain generation")
	debug := flag.Bool("debug", false, "Debug mode: only show subdomain combinations")
	outputFile := flag.String("o", "resultado.txt", "Output file for results")
	workers := flag.Int("workers", 300, "Number of concurrent workers")
	timeout := flag.String("timeout", "3s", "Timeout for HTTP requests (e.g., 5s, 500ms)")
	domainsFile := flag.String("df", "", "Arquivo com lista de domínios (um por linha)")
	flag.Parse()

	if *wordlistFile == "" {
		fmt.Println("Error: You must provide a wordlist file (-w)")
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var domains []string
	if *domainsFile != "" {
		file, err := os.Open(*domainsFile)
		if err != nil {
			fmt.Printf("Erro ao abrir arquivo de domínios: %v\n", err)
			os.Exit(1)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			d := strings.TrimSpace(scanner.Text())
			if d != "" {
				d = strings.TrimPrefix(strings.TrimPrefix(d, "http://"), "https://")
				domains = append(domains, d)
			}
		}
		file.Close()
	} else if *domain != "" {
		d := strings.TrimPrefix(strings.TrimPrefix(*domain, "http://"), "https://")
		domains = []string{d}
	} else {
		fmt.Println("Você deve fornecer um domínio (-d) ou um arquivo de domínios (-df)")
		os.Exit(1)
	}

	timeoutDuration, err := time.ParseDuration(*timeout)
	if err != nil {
		fmt.Printf("Error: Invalid timeout format '%s'. Use formats like '5s', '500ms', etc.\n", *timeout)
		os.Exit(1)
	}

	sc := NewSubCombination()
	sc.wordlist = *wordlistFile
	sc.maxWorkers = *workers
	sc.timeout = timeoutDuration

	start := time.Now()
	results := sc.bruteDomains(domains, *debug)
	elapsed := time.Since(start)

	if !*debug {
		if err := sc.writeOut(results, *outputFile); err != nil {
			fmt.Printf("Error writing results: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n[*] Found %d alive subdomains in %v\n", len(results), elapsed)
		fmt.Printf("[*] Results saved to %s\n", *outputFile)
	}
}
