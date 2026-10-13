# installer/ — Conductino drive installer (CLI)

Ships next to `Conductino.exe` in every release zip (`installer/dist/`).

Build: `go build -o build/bin/Conductino-Installer.exe ./installer`

Run interactive: `build\bin\Conductino-Installer.exe`
Run headless: `Conductino-Installer.exe --app-root D:\Conductino --yes`
Custom binaries: `--src path\to\folder-with-exe`

Layout it creates (update-safe — never deletes `Conductino_Data` contents):

```
<appRoot>\                    e.g. D:\Conductino
  bin\Conductino.exe
  bin\Conductino-Installer.exe
  Conductino_Data\
    config.json / install-info.json / conductino.db
    skills\  cache\extract\  work\summaries\  temporary\extract_dump\
```

`dist/` holds packaged releases (gitignored).
