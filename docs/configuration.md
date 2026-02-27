# Конфигурация и авторизация

## Файл конфигурации

CLI хранит настройки в:

- `~/.xart-cli/config.json`

Поля:

- `api_base_url`
- `user_agent`
- `token`
- `user_id`

## Глобальные флаги

- `--base-url` - переопределить API URL
- `--user-agent` - переопределить User-Agent
- `--token` - передать токен только для текущего запуска
- `--raw` - вывод без pretty JSON

Пример:

```bash
go run . --base-url https://api-s.anixsekai.com --token YOUR_TOKEN release info --id 1
```

## Вход

```bash
go run . auth login --login your_login --password your_password
```

Проверка сессии:

```bash
go run . auth status
go run . auth whoami
```

Выход:

```bash
go run . auth logout
```

Ручная установка токена:

```bash
go run . auth use-token --token YOUR_TOKEN
```

## Регистрация

Шаг 1 - запрос кода:

```bash
go run . auth signup --email you@example.com --login mylogin --password mypassword
```

Шаг 2 - подтверждение кода:

```bash
go run . auth verify-signup --email you@example.com --login mylogin --password mypassword --hash HASH --code 1234
```

## Восстановление пароля

Шаг 1 - запрос кода:

```bash
go run . auth restore --login you@example.com
```

Шаг 2 - подтверждение:

```bash
go run . auth verify-restore --login you@example.com --hash HASH --code 1234 --password new_password
```
