# Справочник команд

## Основные команды

- `auth` - вход, регистрация, восстановление пароля, токен
- `home` - главная лента и категории
- `discover` - discovery-раздел
- `search` - поиск по релизам, профилям, спискам, коллекциям
- `release` - данные релиза, эпизоды, источники, комментарии, оценки
- `bookmarks` - списки просмотра
- `favorites` - избранные релизы
- `history` - история просмотра
- `collections` - коллекции и комментарии коллекций
- `profile` - профиль, друзья, блокировки, настройки
- `api` - низкоуровневые GET/POST/request вызовы
- `ui` - интерактивный терминальный интерфейс
- `watch` - запуск просмотра в локальном плеере

## Примеры

```bash
# Авторизация
go run . auth login --login your_login --password your_password

# Главная
go run . home list last --page 0

# Discovery
go run . discover interesting

# Поиск релизов
go run . search releases --query "naruto" --by name --page 0

# Релиз и эпизоды
go run . release info --id 17443
go run . release episodes --id 17443

# Закладки
go run . bookmarks list --list watching --page 0

# UI
go run . ui

# Просмотр
go run . watch --id 17443
go run . watch --id 17443 --choose-player
```

## Справка по флагам

```bash
go run . --help
go run . <command> --help
go run . <command> <subcommand> --help
```
