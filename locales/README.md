# Translations

Nextclone translations are JSON files in this directory. Each file name is a language code, for example `en.json` or `de.json`.

To add a language:

1. Copy `en.json` to `<language-code>.json`.
2. Translate the values only. Do not change the keys.
3. Keep placeholders like `%s`, `%d`, and `\n` exactly as they are.
4. Run `go run scripts/check-i18n.go` before opening a pull request.

English is the fallback language. If a translation is missing at runtime, Nextclone falls back to English for that key.
