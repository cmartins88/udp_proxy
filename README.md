# UDP Proxy Forwarder

This project acts as a lightweight UDP proxy to forward packets from a game console (e.g., PS5) through a PC so that traffic can be optimized using ExitLag. It captures and redirects UDP traffic, making it appear as if the PC initiated the connection, allowing ExitLag to apply routing optimizations.

---

## 🧩 Features

- Captures and redirects UDP traffic from a console.
- Applies ExitLag optimizations to console traffic.
- Sends responses back to the console to complete the round trip.
- Lightweight, headless operation with logging and log rotation.
- Auto-launches ExitLag and shuts down the proxy when ExitLag closes.

---

## 🔧 Requirements

- Go 1.21 or newer
- [Npcap](https://npcap.com/) installed with WinPcap API compatibility
- ExitLag installed on the PC
- Admin privileges to run the executable (UAC prompt will appear)

---

## 🚀 How to Build

1. Install Go and Npcap.
2. Clone this repository.
3. Create a `config.json` based on `config.example.json`.
4. Run:

```bash
go install github.com/akavel/rsrc@latest
make build_gui
```

> This will embed a manifest that requests admin permissions.

---

## 📁 Configuration (`config.json`)

```json
{
  "ps5_interface": "\\Device\\NPF_{...}",
  "internet_interface": "\\Device\\NPF_{...}",
  "exitlag_path": "C:\\Program Files (x86)\\ExitLag\\ExitLag.exe",
  "log_file": "proxy.log",
  "max_log_size_bytes": 1048576,
  "log_level": "info"
}
```

Use `udp_proxy.exe` logs or Wireshark to find your correct NPF interface strings.

---

## 📦 Output

- `udp_proxy.exe`: compiled binary
- `proxy.log`: logs with rotation
- `*.bak`: archived logs

---

## ✅ License

MIT License