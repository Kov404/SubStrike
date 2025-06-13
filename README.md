# SubStrike - Subdomain Enumeration Tool
**SubStrike** is a high-performance subdomain enumeration tool written in Go, designed for security researchers and penetration testers. It generates and tests subdomains by inserting words from a wordlist into intermediate positions of a given domain (e.g., `api.teste.teste.evil.com`, `teste.api.teste.evil.com`), excluding the domain root and TLD. SubStrike efficiently checks for live subdomains using concurrent HTTP/HTTPS requests, with customizable workers, timeouts, and debug logging.

## Features

- **Targeted Subdomain Generation**: Generates subdomains by inserting words only between existing subdomains (e.g., for `teste.teste.evil.com`, generates `word.teste.teste.evil.com` and `teste.word.teste.evil.com`).
- **Concurrent Testing**: Uses multiple workers to test subdomains via HTTP/HTTPS in parallel, optimizing performance.
- **Customizable Configuration**:
  - Adjustable number of workers (`-t`).
  - Configurable timeout for HTTP requests (`-timeout`).
  - Support for custom wordlists (`-w`) and domain lists (`-f`).
- **Debug Mode**: Detailed logging of generated subdomains and testing process (`-debug`).
- **Progress Tracking**: Real-time progress bar with completion percentage, speed, and estimated time of arrival (ETA).
- **Robust Validation**: Ensures valid inputs for workers, timeout, and required files, with clear error messages.
- **Output**: Saves live subdomains to a specified file (`-o`).

## Installation

### Prerequisites

- **Go**: Version 1.16 or higher recommended.
- **Optional**: A wordlist for subdomain enumeration (e.g., from [SecLists](https://github.com/danielmiessler/SecLists)).

### Steps

Installation
Prerequisites

Go: Version 1.16 or higher recommended.
Optional: A wordlist for subdomain enumeration (e.g., from SecLists).

Steps

Clone the repository:
git clone https://github.com/yourusername/substrike.git
cd substrike


Build the tool:
go build -o substrike combinateDomains.go


(Optional) Move the binary to a directory in your PATH:
sudo mv substrike /usr/local/bin/



Usage
Run SubStrike with the required flags to enumerate subdomains. The basic syntax is:
./substrike -f <domains_file> -w <wordlist_file> -o <output_file> [-t <workers>] [-timeout <ms>] [-debug]

Flags



Flag
Description
Default
Required



-f
File containing domains to enumerate (one per line)
None
Yes


-w
Wordlist file for subdomain generation
/usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt
No


-o
Output file for live subdomains
resultado.txt
No


-t
Number of concurrent workers (1 to 1000)
500
No


-timeout
Timeout for HTTP requests in milliseconds (minimum 100)
1500
No


-debug
Enable debug mode with detailed logs
false
No


Examples
Example 1: Basic Subdomain Enumeration
Enumerate subdomains for api.prod.evil.com using a small wordlist with 50 workers and a 5-second timeout:

Create a file domains.txt:
api.prod.evil.com


Run the command:
./substrike -f domains.txt -w wordlist.txt -o evil_subdomains.txt -t 50 -timeout 5000


Sample wordlist.txt:
admin
auth
staging


Output (example):
🔍 Subdomain Scanner
==================
📁 Domínios: domains.txt
📝 Wordlist: wordlist.txt
💾 Output: evil_subdomains.txt
⚡ Workers: 50
⏱️ Timeout: 5s
🐛 Debug: false

[*] Lendo wordlist...
[*] Carregadas 3 palavras
[*] Lendo domínios...
[*] Carregados 1 domínios
[*] Exemplos de domínios processados:
    - api.prod.evil.com
[*] Total de combinações: 6
[*] Configuração: 50 workers, timeout 5s
[*] Iniciando verificação de subdomínios...
[*] Progresso:
[+] ONLINE: auth.api.prod.evil.com:443 (200)
[*] 🚀 Verificação concluída em tempo recorde!
[*] 📊 Total processado: 6
[*] 🎯 Subdomínios ativos encontrados: 1
[*] 📈 Taxa de sucesso: 16.67%
[*] 💾 Resultados salvos em: evil_subdomains.txt


evil_subdomains.txt:
auth.api.prod.evil.com



Example 2: Advanced Enumeration with Debug Mode
Enumerate subdomains for multiple subdomains of evil.com using a large wordlist, with debug mode enabled, 20 workers, and a 12.5-second timeout:

Create a file evil_domains.txt:
api.prod.evil.com
auth.test.evil.com


Run the command:
./substrike -f evil_domains.txt -w /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt -o evil_results.txt -t 20 -timeout 12500 -debug


Output (example):
🔍 Subdomain Scanner
==================
🐛 Modo DEBUG ativado - Logs salvos em: evil_results_debug.log
📁 Domínios: evil_domains.txt
📝 Wordlist: /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt
💾 Output: evil_results.txt
⚡ Workers: 20
⏱️ Timeout: 12.5s
🐛 Debug: true

[*] Lendo wordlist...
[*] Carregadas 4989 palavras
[*] Lendo domínios...
[*] Carregados 2 domínios
[*] Exemplos de domínios de domínio:
    - api.prod.evil.com
    - auth.test.evil.com
[*] Total de combinações: 19956
[*] Configuração: 20 workers, timeout 12.4s
[*] Iniciando verificação de subdomínios...
[*] Progresso:
[DEBUG] Combinação gerada: admin na posição 1 de api.prod.evil.com = api.admin.prod.evil.com
[GENERATED] Subdomínio criado: api.admin.prod.evil.com
[+] ONLINE: admin.auth.test.evil.com:443 (200)
[*] 🚀 Verificação concluída em tempo recorde!
[*] 📊 Total processado: 19956
[*] 🎯 Subdomínios ativos encontrados: 1
[*] 📈 Taxa de sucesso: 0.005%
[*] 💾 Resultados salvos em: evil_results.txt
[*] 🐛 Logs de debug salvos em: evil_results_debug.log


evil_results.txt:
admin.auth.test.evil.com



Notes

Performance: Use a reasonable number of workers (-t) based on your system's resources. For large wordlists, a smaller number (e.g., 20) may be sufficient.
Timeout: Higher timeouts (e.g., 12500ms) are useful for slow networks but increase total runtime.
Wordlists: Use curated wordlists like those from SecLists for better results.
Debug Mode: Enable -debug to log generated subdomains and detailed testing information. Logs are saved to <output_file>_debug.log.

Contributing
Contributions are welcome! To contribute:

Fork the repository.
Create a new branch (git checkout -b feature/your-feature).
Make your changes and commit (git commit -m "Add your feature").
Push to your branch (git push origin feature/your-feature).
Open a Pull Request.

Please ensure your code follows the existing style and includes appropriate tests.
Ideas for Contributions

Add DNS-based subdomain validation in addition to HTTP/HTTPS checks.
Support for wildcard subdomain filtering.
Export results in JSON or CSV formats.
Implement rate-limiting to avoid overwhelming target servers.

License
This project is licensed under the MIT License. See the LICENSE file for details.
Acknowledgments

Inspired by tools like Sublist3r and Amass.
Thanks to SecLists for providing high-quality wordlists.

Contact
Happy subdomain hunting with SubStrike! 🔍
