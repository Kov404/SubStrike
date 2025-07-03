# SubStrike - Subdomain Enumeration Tool
**SubStrike** is a high-performance subdomain enumeration tool written in Go, designed for security researchers and penetration testers. It generates and tests subdomains by inserting words from a wordlist into intermediate positions of a given domain, excluding the domain root and TLD. SubStrike efficiently checks for live subdomains using concurrent HTTP/HTTPS requests, with customizable workers, timeouts, and debug logging.

## Features

- **Recursive Subdomain Injection**: Inserts words between subdomain levels  (e.g., from `dev.api.target.com`, generates `dev.word.api.target.com` and `dev.api.word.target.com`).
- **Concurrent Testing**: Uses multiple workers to test subdomains via HTTP/HTTPS in parallel, optimizing performance.
- **Customizable Configuration**:
  - Adjustable number of workers (`-workers`).
  - Configurable timeout for HTTP requests (`-timeout`).
  - Support for custom wordlists (`-w`) and domain lists (`-df`).
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


## Usage

Run SubStrike with the required flags to enumerate subdomains. The basic syntax is:

./substrike -df <domains_file> -w <wordlist_file> -o <output_file> [-workers <num>] [-timeout <duration>] [-debug]

### Flags

| Flag        | Description                                               | Default                          | Required |
|-------------|-----------------------------------------------------------|----------------------------------|----------|
| `-df`       | File containing domains to enumerate (one per line)       | None                             | Yes      |
| `-d`        | Single domain to enumerate                                | None                             | Optional |
| `-w`        | Wordlist file for subdomain generation                    | None                             | Yes      |
| `-o`        | Output file for live subdomains                           | `resultado.txt`                  | No       |
| `-workers`  | Number of concurrent workers                              | `300`                            | No       |
| `-timeout`  | Timeout for HTTP requests (e.g., `3s`, `500ms`)           | `3s`                             | No       |
| `-debug`    | Enable debug mode with detailed logs                      | `false`                          | No       |

---

### Examples

#### Example 1: Basic Subdomain Enumeration

Enumerate subdomains for `api.prod.evil.com` using a small wordlist with 50 workers and a 5-second timeout.

**Create a file** `domains.txt`:
**Run the command**:

```bash
./substrike -df domains.txt -w wordlist.txt -o evil_subdomains.txt -workers 50 -timeout 5s
 ```

Notes
Thanks to SecLists for providing high-quality wordlists.

Contact
Happy subdomain hunting with SubStrike! 🔍
