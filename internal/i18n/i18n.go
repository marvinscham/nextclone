package i18n

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	golocale "github.com/jeandeaual/go-locale"
	"github.com/marvinscham/nextclone/locales"
	"golang.org/x/text/language"
)

const (
	SystemLanguage = "system"
	Fallback       = "en"
)

type Localizer struct {
	code     string
	messages map[string]string
}

type Language struct {
	Code string
	Name string
}

func New(setting string) *Localizer {
	code := Resolve(setting)
	messages := readMessages(Fallback)
	if code != Fallback {
		for key, value := range readMessages(code) {
			messages[key] = value
		}
	}
	return &Localizer{code: code, messages: messages}
}

func (l *Localizer) Code() string {
	return l.code
}

func (l *Localizer) T(key string, args ...any) string {
	value := l.messages[key]
	if value == "" {
		value = key
	}
	if len(args) == 0 {
		return value
	}
	return fmt.Sprintf(value, args...)
}

func (l *Localizer) Languages() []Language {
	languages := []Language{{Code: SystemLanguage, Name: l.T("language.system")}}
	for _, code := range Available() {
		languages = append(languages, Language{Code: code, Name: l.T("language." + code)})
	}
	return languages
}

func Available() []string {
	entries, err := locales.FS.ReadDir(".")
	if err != nil {
		return []string{Fallback}
	}

	var codes []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		codes = append(codes, strings.TrimSuffix(entry.Name(), ".json"))
	}
	sort.Strings(codes)
	return codes
}

func Resolve(setting string) string {
	setting = strings.TrimSpace(setting)
	if setting == "" || setting == SystemLanguage {
		return systemLanguage()
	}
	if hasLanguage(setting) {
		return setting
	}
	return Fallback
}

func systemLanguage() string {
	available := Available()
	tags := make([]language.Tag, 0, len(available))
	for _, code := range available {
		tag, err := language.Parse(code)
		if err == nil {
			tags = append(tags, tag)
		}
	}
	matcher := language.NewMatcher(tags)

	locales, err := golocale.GetLocales()
	if err != nil || len(locales) == 0 {
		locale, localeErr := golocale.GetLocale()
		if localeErr != nil || locale == "" {
			return Fallback
		}
		locales = []string{locale}
	}

	preferences := make([]language.Tag, 0, len(locales))
	for _, locale := range locales {
		locale = strings.ReplaceAll(locale, "_", "-")
		tag, err := language.Parse(locale)
		if err == nil {
			preferences = append(preferences, tag)
		}
	}
	_, index, confidence := matcher.Match(preferences...)
	if confidence == language.No || index < 0 || index >= len(available) {
		return Fallback
	}
	return available[index]
}

func hasLanguage(code string) bool {
	for _, available := range Available() {
		if code == available {
			return true
		}
	}
	return false
}

func readMessages(code string) map[string]string {
	data, err := locales.FS.ReadFile(code + ".json")
	if err != nil {
		return map[string]string{}
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return map[string]string{}
	}
	return messages
}
