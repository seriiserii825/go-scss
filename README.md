# go-scss

Go-версия [py-scss](../../python/py-scss). Следит за буфером обмена: CSS,
скопированный из Figma, приводится к виду для проекта и кладётся обратно.

```
cd ~/Sites/my-project   # корень проекта, где лежит package.json
go-scss                 # или go-scss --to-rem
```

## Что делает

По порядку:

1. `letter-spacing` px/% -> `em`, `line-height` px/% -> без единиц
   (относительно первого `font-size` в тексте).
2. `--to-rem`: `var(--x, 16px)` и `16px` -> `1.6rem` (строки с `border: 1px`,
   `linear-gradient`, `&` не трогаются).
3. Убирает комментарии `/* ... */`.
4. Удаляет строки с `font-feature-settings`, `text-overflow`, `overflow`,
   `font-kerning`, `leading-trim`, `text-edge`.
5. Замены из `projects/<имя папки проекта>.txt`.
6. `color` — перед `background`, `text-align` и `text-shadow` — перед `color`.
7. Удаляет пустые строки.

Обрабатывается только текст с `:`.

## Файл переменных

`~/Documents/go/go-scss/projects/<имя папки проекта>.txt`, по строке на замену:

```
#F2F2F2|var(--text)
var(--Red, #E0373C)|var(--accent)
Instrument Sans|delete
```

`delete` удаляет строку целиком. Регистр при поиске не важен, замена
пишется в нижнем регистре. Файл перечитывается на каждое копирование.
Если файла нет — создаётся пустой, и скрипт просит его заполнить.

## Установка

```
go install github.com/seriiserii825/go-scss@latest
```

Нужны `xclip` (X11) или `wl-clipboard` (Wayland), `libnotify` — по желанию.

## Разработка

go-libs подключается по тегу из `go.mod`. Обновить:

```
go get github.com/seriiserii825/go-libs@latest && go mod tidy
go vet ./... && go test ./...
```
