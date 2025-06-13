package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Global variables for logging
var (
	debugLogger *log.Logger
	debugFile   *os.File
	isDebugMode bool
)

// initDebugLogging inicializa o sistema de logging de debug
func initDebugLogging(filename string) error {
	var err error
	debugFile, err = os.Create(filename)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo de debug: %v", err)
	}

	debugLogger = log.New(debugFile, "", log.LstdFlags|log.Lmicroseconds)
	isDebugMode = true

	debugLogger.Println("=== DEBUG LOG INICIADO ===")
	debugLogger.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))
	debugLogger.Println("===========================")

	return nil
}

// debugLog escreve no log de debug se o modo debug estiver ativo
func debugLog(format string, args ...interface{}) {
	if isDebugMode {
		// Formata a mensagem
		message := fmt.Sprintf(format, args...)

		// Escreve no arquivo de debug, se disponível
		if debugLogger != nil {
			debugLogger.Println(message)
		}

		// Escreve na tela (stdout)
		fmt.Println("[DEBUG] " + message)
	}
}

// readWordlist lê uma lista de palavras de um arquivo
func readWordlist(file string) ([]string, error) {
	debugLog("Iniciando leitura da wordlist: %s", file)

	f, err := os.Open(file)
	if err != nil {
		debugLog("ERRO ao abrir wordlist %s: %v", file, err)
		return nil, fmt.Errorf("falha ao abrir wordlist: %v", err)
	}
	defer f.Close()

	var words []string
	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			words = append(words, word)
			if len(words) <= 10 { // Log apenas as primeiras 10 palavras
				debugLog("Palavra carregada [%d]: %s", len(words), word)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		debugLog("ERRO ao ler wordlist: %v", err)
		return nil, fmt.Errorf("erro ao ler wordlist: %v", err)
	}

	debugLog("Wordlist carregada com sucesso: %d palavras de %d linhas", len(words), lineCount)
	return words, nil
}

// cleanDomain remove prefixos http://, https:// e www. do domínio
func cleanDomain(domain string) string {
	originalDomain := domain

	// Remove https://
	if strings.HasPrefix(domain, "https://") {
		domain = strings.TrimPrefix(domain, "https://")
	}

	// Remove http://
	if strings.HasPrefix(domain, "http://") {
		domain = strings.TrimPrefix(domain, "http://")
	}

	// Remove qualquer barra no final
	domain = strings.TrimSuffix(domain, "/")

	if originalDomain != domain {
		debugLog("Domínio limpo: '%s' -> '%s'", originalDomain, domain)
	}

	return domain
}

// readDomains lê uma lista de domínios de um arquivo e limpa os prefixos
func readDomains(file string) ([]string, error) {
	debugLog("Iniciando leitura dos domínios: %s", file)

	f, err := os.Open(file)
	if err != nil {
		debugLog("ERRO ao abrir arquivo de domínios %s: %v", file, err)
		return nil, fmt.Errorf("falha ao abrir arquivo de domínios: %v", err)
	}
	defer f.Close()

	var domains []string
	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" {
			// Limpa o domínio removendo prefixos
			cleanedDomain := cleanDomain(domain)
			if cleanedDomain != "" {
				domains = append(domains, cleanedDomain)
				if len(domains) <= 10 { // Log apenas os primeiros 10 domínios
					debugLog("Domínio carregado [%d]: %s", len(domains), cleanedDomain)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		debugLog("ERRO ao ler domínios: %v", err)
		return nil, fmt.Errorf("erro ao ler domínios: %v", err)
	}

	debugLog("Domínios carregados com sucesso: %d domínios de %d linhas", len(domains), lineCount)
	return domains, nil
}

// generateSubdomains gera subdomínios inserindo palavras apenas entre os subdomínios existentes
func generateSubdomains(domain string, word string) []string {
	// Divide o domínio em partes (ex.: teste.teste.testvuln.com -> [teste, teste, testvuln.com])
	parts := strings.Split(domain, ".")
	if len(parts) < 3 {
		// Se o domínio tiver menos de 3 partes (ex.: testvuln.com), gera apenas um subdomínio com a palavra no início
		subdomain := fmt.Sprintf("%s.%s", word, domain)
		if isDebugMode {
			fmt.Printf("[GENERATED] Subdomínio criado: %s\n", subdomain)
		}
		debugLog("Combinação gerada: %s + %s = %s", word, domain, subdomain)
		return []string{subdomain}
	}

	var subdomains []string
	// Itera até o penúltimo nível (exclui o último nível, testvuln.com)
	for i := 0; i < len(parts)-1; i++ {
		// Constrói o subdomínio inserindo a palavra na posição i
		var newParts []string
		newParts = append(newParts, parts[:i]...)
		newParts = append(newParts, word)
		newParts = append(newParts, parts[i:]...)
		subdomain := strings.Join(newParts, ".")
		if isDebugMode {
			fmt.Printf("[GENERATED] Subdomínio criado: %s\n", subdomain)
		}
		debugLog("Combinação gerada: %s na posição %d de %s = %s", word, i, domain, subdomain)
		subdomains = append(subdomains, subdomain)
	}
	return subdomains
}

// Global HTTP clients para reutilização
var (
	httpsClient *http.Client
	httpClient  *http.Client
	clientOnce  sync.Once
)

// initClients inicializa os clients HTTP globais
func initClients(timeout time.Duration) {
	clientOnce.Do(func() {
		tr := &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   100,
			MaxConnsPerHost:       200,
			IdleConnTimeout:       10 * time.Second,
			TLSHandshakeTimeout:   timeout / 2,
			ResponseHeaderTimeout: timeout / 2,
			DisableKeepAlives:     false,
		}

		httpsClient = &http.Client{
			Transport: tr,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		httpClient = &http.Client{
			Transport: tr,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	})
}

// isAlive verifica se um subdomínio está ativo usando goroutines paralelas
func isAlive(domain string, timeout time.Duration) (bool, string) {
	initClients(timeout)

	debugLog("Testando subdomínio: %s", domain)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []string

	// Testa HTTPS e HTTP em paralelo
	schemes := []struct {
		url    string
		port   string
		client *http.Client
	}{
		{"https://" + domain, "443", httpsClient},
		{"http://" + domain, "80", httpClient},
	}

	for _, s := range schemes {
		wg.Add(1)
		go func(url, port string, client *http.Client) {
			defer wg.Done()

			debugLog("Testando URL: %s", url)
			resp, err := client.Get(url)
			if err == nil {
				statusCode := resp.StatusCode
				resp.Body.Close()
				debugLog("Resposta de %s: Status %d", url, statusCode)
				if statusCode < 400 {
					mu.Lock()
					result := fmt.Sprintf("%s:%s (%d)", domain, port, statusCode)
					results = append(results, result)
					debugLog("✅ SUCESSO: %s", result)
					mu.Unlock()
				}
			} else {
				debugLog("❌ ERRO em %s: %v", url, err)
			}
		}(s.url, s.port, s.client)
	}

	wg.Wait()

	if len(results) > 0 {
		finalResult := strings.Join(results, " | ")
		debugLog("🎯 Subdomínio ATIVO encontrado: %s", finalResult)
		return true, finalResult
	}

	debugLog("💤 Subdomínio inativo: %s", domain)
	return false, ""
}

// worker processa subdomínios em lotes
func worker(jobs <-chan string, results chan<- string, timeout time.Duration, wg *sync.WaitGroup, processed *int64) {
	defer wg.Done()

	for domain := range jobs {
		if alive, info := isAlive(domain, timeout); alive {
			fmt.Printf("[+] ONLINE: %s\n", info)
			results <- domain
		}
		atomic.AddInt64(processed, 1)
	}
}

// progressBar cria uma barra de progresso visual
func progressBar(current, total int64) string {
	percent := float64(current) / float64(total) * 100
	barLength := 50
	filled := int(percent / 100 * float64(barLength))

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "="
		} else if i == filled {
			bar += ">"
		} else {
			bar += " "
		}
	}
	bar += "]"

	return fmt.Sprintf("%s %.2f%% (%d/%d)", bar, percent, current, total)
}

func main() {
	// Definição dos flags de linha de comando
	var domain = flag.String("d", "", "Domínio único para enumerar (ex.: api.prod.evil.com)")
	var domainsFile = flag.String("f", "", "Arquivo com lista de domínios")
	var wordlistFile = flag.String("w", "/usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt", "Arquivo de wordlist")
	var outputFile = flag.String("o", "resultado.txt", "Arquivo de saída")
	var workers = flag.Int("t", 500, "Número de workers/threads")
	var timeoutMs = flag.Int("timeout", 1500, "Timeout em milissegundos")
	var debug = flag.Bool("debug", false, "Ativar modo debug com logs detalhados")

	flag.Parse()

	// Verifica argumentos posicionais não utilizados
	if flag.NArg() > 0 {
		fmt.Printf("❌ Erro: Argumentos posicionais não suportados: %v\n", flag.Args())
		fmt.Println("Use flags como -d, -f, -w, -o, -t, -timeout, -debug para especificar valores")
		fmt.Println("\nUso:")
		fmt.Println("  -d <domínio>    Domínio único para enumerar (ex.: api.prod.evil.com)")
		fmt.Println("  -f <arquivo>    Arquivo com lista de domínios")
		fmt.Println("  -w <arquivo>    Arquivo de wordlist (padrão: /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt)")
		fmt.Println("  -o <arquivo>    Arquivo de saída (padrão: resultado.txt)")
		fmt.Println("  -t <número>     Número de workers/threads (padrão: 500, mínimo: 1, máximo: 1000)")
		fmt.Println("  -timeout <ms>   Timeout em milissegundos (padrão: 1500, mínimo: 100)")
		fmt.Println("  -debug          Ativar modo debug com logs detalhados")
		fmt.Println("\nExemplo:")
		fmt.Println("  go run combinateDomains.go -d api.prod.evil.com -w wordlist.txt -o resultados.txt -t 20 -timeout 12500 -debug")
		fmt.Println("  go run combinateDomains.go -f dominios.txt -w wordlist.txt -o resultados.txt -t 20 -timeout 12500 -debug")
		os.Exit(1)
	}

	// Validação das flags
	if *domain == "" && *domainsFile == "" {
		fmt.Println("❌ Erro: Um dos parâmetros -d ou -f é obrigatório!")
		fmt.Println("\nUso:")
		fmt.Println("  -d <domínio>    Domínio único para enumerar (ex.: api.prod.evil.com)")
		fmt.Println("  -f <arquivo>    Arquivo com lista de domínios")
		fmt.Println("  -w <arquivo>    Arquivo de wordlist (padrão: /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt)")
		fmt.Println("  -o <arquivo>    Arquivo de saída (padrão: resultado.txt)")
		fmt.Println("  -t <número>     Número de workers/threads (padrão: 500, mínimo: 1, máximo: 1000)")
		fmt.Println("  -timeout <ms>   Timeout em milissegundos (padrão: 1500, mínimo: 100)")
		fmt.Println("  -debug          Ativar modo debug com logs detalhados")
		fmt.Println("\nExemplo:")
		fmt.Println("  go run combinateDomains.go -d api.prod.evil.com -w wordlist.txt -o resultados.txt -t 20 -timeout 12500 -debug")
		fmt.Println("  go run combinateDomains.go -f dominios.txt -w wordlist.txt -o resultados.txt -t 20 -timeout 12500 -debug")
		os.Exit(1)
	}
	if *domain != "" && *domainsFile != "" {
		fmt.Println("❌ Erro: Não é possível usar -d e -f ao mesmo tempo!")
		os.Exit(1)
	}
	if *workers < 1 || *workers > 1000 {
		fmt.Printf("❌ Erro: Número de workers inválido (%d). Deve estar entre 1 e 1000.\n", *workers)
		os.Exit(1)
	}
	if *timeoutMs < 100 {
		fmt.Printf("❌ Erro: Timeout inválido (%d ms). Deve ser maior ou igual a 100 ms.\n", *timeoutMs)
		os.Exit(1)
	}

	// Inicializa debug logging se solicitado
	if *debug {
		debugLogFile := strings.TrimSuffix(*outputFile, ".txt") + "_debug.log"
		if err := initDebugLogging(debugLogFile); err != nil {
			fmt.Printf("❌ Erro ao inicializar debug logging: %v\n", err)
			os.Exit(1)
		}
		defer debugFile.Close()
		fmt.Printf("🐛 Modo DEBUG ativado - Logs salvos em: %s\n", debugLogFile)
	}

	timeout := time.Duration(*timeoutMs) * time.Millisecond
	maxWorkers := *workers
	batchSize := maxWorkers * 10 // Ajuste dinâmico do batchSize

	fmt.Println("🔍 Subdomain Scanner")
	fmt.Println("==================")
	if *domain != "" {
		fmt.Printf("🌐 Domínio: %s\n", *domain)
	} else {
		fmt.Printf("📁 Domínios: %s\n", *domainsFile)
	}
	fmt.Printf("📝 Wordlist: %s\n", *wordlistFile)
	fmt.Printf("💾 Output: %s\n", *outputFile)
	fmt.Printf("⚡ Workers: %d\n", maxWorkers)
	fmt.Printf("⏱️  Timeout: %v\n", timeout)
	fmt.Printf("🐛 Debug: %v\n", *debug)
	fmt.Println()

	debugLog("=== CONFIGURAÇÕES ===")
	if *domain != "" {
		debugLog("Domínio: %s", *domain)
	} else {
		debugLog("Domínios: %s", *domainsFile)
	}
	debugLog("Wordlist: %s", *wordlistFile)
	debugLog("Output: %s", *outputFile)
	debugLog("Workers: %d", maxWorkers)
	debugLog("Timeout: %v", timeout)
	debugLog("====================")

	fmt.Println("[*] Lendo wordlist...")
	words, err := readWordlist(*wordlistFile)
	if err != nil {
		fmt.Println("Erro ao ler wordlist:", err)
		os.Exit(1)
	}
	fmt.Printf("[*] Carregadas %d palavras\n", len(words))

	// Lê domínios (de -d ou -f)
	fmt.Println("[*] Lendo domínios...")
	var domains []string
	if *domain != "" {
		domains = []string{cleanDomain(*domain)}
		fmt.Println("[*] Usando domínio fornecido diretamente")
	} else {
		domains, err = readDomains(*domainsFile)
		if err != nil {
			fmt.Println("Erro ao ler domínios:", err)
			os.Exit(1)
		}
	}
	fmt.Printf("[*] Carregados %d domínios\n", len(domains))

	// Debug: mostra alguns domínios limpos
	fmt.Println("[*] Exemplos de domínios processados:")
	for i, domain := range domains {
		if i < 3 { // Mostra apenas os 3 primeiros
			fmt.Printf("    - %s\n", domain)
		}
	}

	// Calcula total de combinações
	totalCombinations := int64(0)
	for _, domain := range domains {
		parts := strings.Split(domain, ".")
		if len(parts) >= 2 {
			totalCombinations += int64(len(words) * (len(parts) - 1))
		} else {
			totalCombinations += int64(len(words))
		}
	}
	fmt.Printf("[*] Total de combinações: %d\n", totalCombinations)
	fmt.Printf("[*] Configuração: %d workers, timeout %v\n", maxWorkers, timeout)

	debugLog("Total de combinações que serão testadas: %d", totalCombinations)
	debugLog("Iniciando processo de geração de combinações...")

	// Inicializa clients HTTP globais
	initClients(timeout)

	// Cria arquivo de saída
	outputFileHandle, err := os.Create(*outputFile)
	if err != nil {
		fmt.Println("Erro ao criar arquivo de saída:", err)
		os.Exit(1)
	}
	defer outputFileHandle.Close()

	// Configura canais e workers
	jobs := make(chan string, batchSize)
	results := make(chan string, batchSize)
	var wg sync.WaitGroup
	var processed int64
	var aliveCount int64

	// Inicia workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(jobs, results, timeout, &wg, &processed)
	}

	// Goroutine para processar resultados
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		writer := bufio.NewWriter(outputFileHandle)
		defer writer.Flush()

		for result := range results {
			writer.WriteString(result + "\n")
			if atomic.AddInt64(&aliveCount, 1)%10 == 0 {
				writer.Flush() // Atualizar a cada 10 resultados
			}
		}
	}()

	// Goroutine para mostrar progresso
	var progressWg sync.WaitGroup
	progressWg.Add(1)
	go func() {
		defer progressWg.Done()
		ticker := time.NewTicker(1 * time.Second) // Atualização mais frequente
		defer ticker.Stop()

		startTime := time.Now()

		for {
			select {
			case <-ticker.C:
				currentProcessed := atomic.LoadInt64(&processed)
				currentAlive := atomic.LoadInt64(&aliveCount)

				if currentProcessed >= totalCombinations {
					return
				}

				// Calcula velocidade (requisições por segundo)
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					rps := float64(currentProcessed) / elapsed

					// Calcula ETA
					remaining := totalCombinations - currentProcessed
					var eta time.Duration
					if rps > 0 {
						eta = time.Duration(float64(remaining)/rps) * time.Second
					}

					fmt.Printf("\r[*] %s | Encontrados: %d | Velocidade: %.0f req/s | ETA: %v",
						progressBar(currentProcessed, totalCombinations),
						currentAlive, rps, eta.Round(time.Second))
				}
			}
		}
	}()

	fmt.Println("[*] Iniciando verificação de subdomínios...")
	fmt.Println("[*] Progresso:")

	// Envia todos os jobs de uma vez para maximizar throughput
	go func() {
		defer close(jobs)
		combinationCount := 0
		debugLog("Iniciando geração de combinações...")

		for _, domain := range domains {
			for _, word := range words {
				// Gera múltiplos subdomínios para cada palavra e domínio
				subdomains := generateSubdomains(domain, word)
				for _, subdomain := range subdomains {
					combinationCount++
					// Log apenas as primeiras 20 combinações para não sobrecarregar
					if combinationCount <= 20 {
						debugLog("Enviando combinação #%d: %s", combinationCount, subdomain)
					} else if combinationCount == 21 {
						debugLog("... (logging de combinações individuais pausado para performance)")
					}
					jobs <- subdomain
				}
			}
		}
		debugLog("Todas as %d combinações foram enviadas para processamento", combinationCount)
	}()

	// Aguarda workers terminarem
	wg.Wait()

	// Fecha canal de resultados e aguarda processamento
	close(results)
	resultWg.Wait()

	// Para a goroutine de progresso
	progressWg.Wait()

	finalAlive := atomic.LoadInt64(&aliveCount)

	debugLog("=== RESULTADO FINAL ===")
	debugLog("Total processado: %d", totalCombinations)
	debugLog("Subdomínios ativos: %d", finalAlive)
	debugLog("Taxa de sucesso: %.2f%%", float64(finalAlive)/float64(totalCombinations)*100)
	debugLog("Scan finalizado com sucesso!")
	debugLog("=======================")

	fmt.Printf("\n\n[*] 🚀 Verificação concluída em tempo recorde!\n")
	fmt.Printf("[*] 📊 Total processado: %d\n", totalCombinations)
	fmt.Printf("[*] 🎯 Subdomínios ativos encontrados: %d\n", finalAlive)
	fmt.Printf("[*] 📈 Taxa de sucesso: %.2f%%\n", float64(finalAlive)/float64(totalCombinations)*100)
	fmt.Printf("[*] 💾 Resultados salvos em: %s\n", *outputFile)

	if *debug {
		debugLogFile := strings.TrimSuffix(*outputFile, ".txt") + "_debug.log"
		fmt.Printf("[*] 🐛 Logs de debug salvos em: %s\n", debugLogFile)
	}
}
