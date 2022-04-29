package mars

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/robfig/config"
)

const (
	CurrentLocaleRenderArg = "currentLocale" // The key for the current locale render arg value

	messageFilesDirectory = "messages"
	messageFilePattern    = `^\w+\.[a-zA-Z]{2}$`
	unknownValueFormat    = "??? %s ???"
	defaultLanguageOption = "i18n.default_language"
	localeCookieConfigKey = "i18n.cookie"
)

var (
	// All currently loaded message configs.
	messages map[string]*config.Config
)

// Return all currently loaded message languages.
func MessageLanguages() []string {
	languages := make([]string, len(messages))
	i := 0
	for language := range messages {
		languages[i] = language
		i++
	}
	return languages
}

// Perform a message look-up for the given locale and message using the given arguments.
//
// When either an unknown locale or message is detected, a specially formatted string is returned.
func Message(locale, message string, args ...interface{}) string {
	language, region := parseLocale(locale)

	messageConfig, knownLanguage := messages[language]
	if !knownLanguage {
		TRACE.Printf("Unsupported language for locale '%s' and message '%s', trying default language", locale, message)

		if defaultLanguage, found := Config.String(defaultLanguageOption); found {
			TRACE.Printf("Using default language '%s'", defaultLanguage)

			messageConfig, knownLanguage = messages[defaultLanguage]
			if !knownLanguage {
				WARN.Printf("Unsupported default language for locale '%s' and message '%s'", defaultLanguage, message)
				return fmt.Sprintf(unknownValueFormat, message)
			}
		} else {
			WARN.Printf("Unable to find default language option (%s); messages for unsupported locales will never be translated", defaultLanguageOption)
			return fmt.Sprintf(unknownValueFormat, message)
		}
	}

	// This works because unlike the goconfig documentation suggests it will actually
	// try to resolve message in DEFAULT if it did not find it in the given section.
	value, error := messageConfig.String(region, message)
	if error != nil {
		WARN.Printf("Unknown message '%s' for locale '%s'", message, locale)
		return fmt.Sprintf(unknownValueFormat, message)
	}

	if len(args) > 0 {
		TRACE.Printf("Arguments detected, formatting '%s' with %v", value, args)
		value = fmt.Sprintf(value, args...)
	}

	return value
}

// MessageHTML performs a message look-up for the given locale and message using the given arguments
// and guarantees, that safe HTML is always returned.
func MessageHTML(locale, key string, args ...interface{}) template.HTML {
	if !strings.HasSuffix(key, ".html") && !strings.HasSuffix(key, "_html") {
		return template.HTML(html.EscapeString(Message(locale, key, args...)))
	}

	safeArgs := make([]interface{}, len(args))
	for idx, arg := range args {
		switch val := arg.(type) {
		case template.HTML:
			safeArgs[idx] = val
		case string:
			safeArgs[idx] = html.EscapeString(val)
		case fmt.Stringer:
			safeArgs[idx] = html.EscapeString(val.String())
		case []byte:
			safeArgs[idx] = []byte(html.EscapeString(string(val)))
		case bool:
			safeArgs[idx] = val
		case float32:
			safeArgs[idx] = val
		case float64:
			safeArgs[idx] = val
		case complex64:
			safeArgs[idx] = val
		case complex128:
			safeArgs[idx] = val
		case int:
			safeArgs[idx] = val
		case int8:
			safeArgs[idx] = val
		case int16:
			safeArgs[idx] = val
		case int32:
			safeArgs[idx] = val
		case int64:
			safeArgs[idx] = val
		case uint:
			safeArgs[idx] = val
		case uint8:
			safeArgs[idx] = val
		case uint16:
			safeArgs[idx] = val
		case uint32:
			safeArgs[idx] = val
		case uint64:
			safeArgs[idx] = val
		case uintptr:
			safeArgs[idx] = val
		default:
			safeArgs[idx] = html.EscapeString(fmt.Sprint(val))
		}
	}

	return template.HTML(Message(locale, key, safeArgs...))
}

func parseLocale(locale string) (language, region string) {
	if strings.Contains(locale, "-") {
		languageAndRegion := strings.Split(locale, "-")
		return languageAndRegion[0], languageAndRegion[1]
	}

	return locale, ""
}

// Recursively read and cache all available messages from all message files on the given path.
func loadMessages(path string) {
	messages = make(map[string]*config.Config)

	if error := filepath.Walk(path, loadMessageFile); error != nil && !os.IsNotExist(error) {
		ERROR.Println("Error reading messages files:", error)
	}
}

// Load a single message file
func loadMessageFile(path string, info os.FileInfo, osError error) error {
	if osError != nil {
		return osError
	}
	if info.IsDir() {
		return nil
	}

	if matched, _ := regexp.MatchString(messageFilePattern, info.Name()); matched {
		if config, error := parseMessagesFile(path); error != nil {
			return error
		} else {
			locale := parseLocaleFromFileName(info.Name())

			// If we have already parsed a message file for this locale, merge both
			if _, exists := messages[locale]; exists {
				messages[locale].Merge(config)
				TRACE.Printf("Successfully merged messages for locale '%s'", locale)
			} else {
				messages[locale] = config
			}

			TRACE.Println("Successfully loaded messages from file", info.Name())
		}
	} else {
		TRACE.Printf("Ignoring file %s because it did not have a valid extension", info.Name())
	}

	return nil
}

func parseMessagesFile(path string) (messageConfig *config.Config, error error) {
	messageConfig, error = config.ReadDefault(path)
	return
}

func parseLocaleFromFileName(file string) string {
	extension := filepath.Ext(file)[1:]
	return strings.ToLower(extension)
}

func init() {
	OnAppStart(func() {
		loadMessages(filepath.Join(BasePath, messageFilesDirectory))
	})
}

func I18nFilter(c *Controller, fc []Filter) {
	if foundCookie, cookieValue := hasLocaleCookie(c.Request); foundCookie {
		TRACE.Printf("Found locale cookie value: %s", cookieValue)
		setCurrentLocaleControllerArguments(c, cookieValue)
	} else if foundHeader, headerValue := hasAcceptLanguageHeader(c.Request); foundHeader {
		TRACE.Printf("Found Accept-Language header value: %s", headerValue)
		setCurrentLocaleControllerArguments(c, headerValue)
	} else {
		TRACE.Println("Unable to find locale in cookie or header, using empty string")
		setCurrentLocaleControllerArguments(c, "")
	}
	fc[0](c, fc[1:])
}

// Set the current locale controller argument (CurrentLocaleControllerArg) with the given locale.
func setCurrentLocaleControllerArguments(c *Controller, locale string) {
	c.Request.Locale = locale
	c.RenderArgs[CurrentLocaleRenderArg] = locale
}

// Determine whether the given request has valid Accept-Language value.
//
// Assumes that the accept languages stored in the request are sorted according to quality, with top
// quality first in the slice.
func hasAcceptLanguageHeader(request *Request) (bool, string) {
	if request.AcceptLanguages != nil && len(request.AcceptLanguages) > 0 {
		return true, removeAllWhitespace(request.AcceptLanguages[0].Language)
	}

	return false, ""
}

// Determine whether the given request has a valid language cookie value.
func hasLocaleCookie(request *Request) (bool, string) {
	if request != nil && request.Cookies() != nil {
		name := Config.StringDefault(localeCookieConfigKey, CookiePrefix+"_LANG")
		if cookie, error := request.Cookie(name); error == nil {
			return true, removeAllWhitespace(cookie.Value)
		} else {
			TRACE.Printf("Unable to read locale cookie with name '%s': %s", name, error.Error())
		}
	}

	return false, ""
}
