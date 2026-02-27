# xart-cli

Консольный клиент Anixart/Xart на Go.

CLI повторяет функции сайта `xart-rose.vercel.app` через API `api-s.anixsekai.com`:

- авторизация;
- главная лента и фильтры;
- discovery-раздел;
- поиск;
- релизы, эпизоды, плеерные источники, голосование, комментарии;
- закладки, избранное, история;
- коллекции и комментарии коллекций;
- профиль, друзья, блокировки, настройки.

Для редко используемых endpoint-ов есть универсальные команды `xart api get/post/request`.

## Установка и запуск

```bash
go mod tidy
go run . --help
```

Или собрать бинарник:

```bash
go build -o xart.exe .
```

## Быстрый старт

```bash
# Вход и сохранение токена в ~/.xart-cli/config.json
go run . auth login --login your_login --password your_password

# Домашняя категория "последнее"
go run . home list last --page 0

# Поиск по релизам
go run . search releases --query "наруто" --by name --page 0

# Информация о релизе
go run . release info --id 18174

# Эпизоды/источники/стрим
go run . release episodes --id 18174
go run . release sources --id 18174 --voiceover-id 1
go run . release stream --id 18174 --voiceover-id 1 --source-id 2

# Смотреть видео в плеере из терминала
go run . watch --id 17443
```

## Конфиг

Файл конфигурации: `~/.xart-cli/config.json`

Поля:

- `api_base_url` (по умолчанию `https://api-s.anixsekai.com`);
- `user_agent` (как в веб-клиенте Xart);
- `token`;
- `user_id`.

Переопределение в рантайме:

```bash
go run . --base-url https://api-s.anixsekai.com --token YOUR_TOKEN release info --id 1
```

## Справка по командам

```bash
go run . --help
go run . release --help
go run . profile --help
go run . api --help
```


## UI Mode

Run interactive title cards UI:

```bash
go run . ui
```

Для просмотра видео нужен установленный плеер: `mpv`, `vlc` или `ffplay`.

Keys:
- `h/j/k/l` or arrows: move selection
- `Enter`: open title details
- `w`: watch selected title in local player
- `f`: toggle favorite
- `0..5`: set watch list status
- `b`: open bookmarks section
- `g`: back to home section
- `Tab` or `[` `]`: switch category
- `n` / `p`: next / previous page
- `esc`: back from details
- in details: `j/k` or `↑/↓` scroll content
- `q`: quit
- `i`: login form (sign in)
- `u`: sign up form (register)
- `o`: logout
- In auth form: `Tab`/`Shift+Tab` switch field, `Enter` submit, `Esc` close form

Bookmarks categories in UI:
- `favorite`, `watching`, `planned`, `watched`, `delayed`, `abandoned`
