# Гайды по подключению VPN

Положите PDF-файлы с инструкциями в эту папку:

| Файл              | Платформа |
|-------------------|-----------|
| `ios.pdf`         | iOS       |
| `android.pdf`     | Android   |
| `windows.pdf`     | Windows   |
| `macos.pdf`       | macOS     |
| `linux.pdf`       | Linux     |

Бот отправит соответствующий PDF, когда пользователь выберет платформу.

Чтобы добавить новую платформу — добавьте файл и зарегистрируйте платформу
в `FSProvider.platforms` (файл `internal/guide/fs_provider.go`).
