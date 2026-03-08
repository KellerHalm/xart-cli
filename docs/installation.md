# Установка

## Требования

- Go `1.25+`
- доступ к интернету для API-запросов
- опционально: установленный плеер `mpv`, `vlc` или `ffplay` для команды `watch`

## Клонирование и запуск

```bash
git clone <repo-url>
cd xart-cli
go mod tidy
go run . --help
```

## Сборка

```bash
go build -o xart.exe .
```

## Установка плеера

### Windows

Через Chocolatey:

```powershell
choco install mpv -y
```

или

```powershell
choco install vlc -y
```

Проверка:

```powershell
mpv --version
```

### Linux

Debian/Ubuntu:

```bash
sudo apt install mpv
# или
sudo apt install vlc
```

### macOS

```bash
brew install mpv
# или
brew install --cask vlc
```

## Проверка проекта

```bash
go test ./...
```
