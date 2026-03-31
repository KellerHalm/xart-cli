#  оманда просмотра через watch

 оманда `watch` выбирает доступный эпизод через API и запускает его в локальном плеере.

## Ѕазовый запуск

```bash
go run . watch --id 17443
```

ѕо умолчанию:

- выбираетс€ перва€ доступна€ озвучка;
- выбираетс€ первый доступный источник;
- выбираетс€ последний доступный эпизод;
- плеер выбираетс€ автоматически в пор€дке: `mpv` -> `vlc` -> `ffplay`.

## ‘лаги

- `--id` (об€зательно) - ID релиза
- `--voiceover-id` - ID озвучки
- `--source-id` - ID источника
- `--episode` - позици€ эпизода
- `--player` - плеер (`mpv|vlc|ffplay`) или путь к бинарнику
- `--choose-player` - показать интерактивный выбор плеера перед запуском
- `--player-arg` - доп. аргумент плеера (можно повтор€ть)
- `--print-url` - вывести только URL эпизода
- `--no-progress` - не вызывать `history/add` и `episode/watch`

## ѕримеры

¬ывести только URL:

```bash
go run . watch --id 17443 --print-url
```

¬ыбрать озвучку/источник/эпизод:

```bash
go run . watch --id 17443 --voiceover-id 10 --source-id 10 --episode 120
```

«апустить `mpv` с аргументами:

```bash
go run . watch --id 17443 --player mpv --player-arg=--fullscreen --player-arg=--volume=50
```

»нтерактивно выбрать плеер:

```bash
go run . watch --id 17443 --choose-player
```

## ќграничени€

-  оманда зависит от того, какие данные вернет API дл€ релиза.
- ƒл€ части источников может вернутьс€ iframe-URL; надежнее всего работает `mpv`.
- ≈сли плеер не найден, команда завершитс€ ошибкой и подскажет доступные варианты.

## gomp TUI

`watch` can also launch the new `gomp` TUI player:

```bash
go run . watch --id 17443 --gomp
```

In this mode:

- voiceover, source, and episode are chosen inside the TUI;
- playback uses the `mpv` backend from `gomp`;
- `--player` may be used only to point to an `mpv` executable;
- `--choose-player` and `--print-url` are not supported together with `--gomp`.

