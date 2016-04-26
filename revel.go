package mars

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/agtorre/gocolorize"
	"github.com/robfig/config"
)

const (
	MarsImportPath     = "github.com/roblillack/mars"
	defaultLoggerFlags = log.Ldate | log.Ltime | log.Lshortfile
)

type marsLogs struct {
	c gocolorize.Colorize
	w io.Writer
}

func (r *marsLogs) Write(p []byte) (n int, err error) {
	return r.w.Write([]byte(r.c.Paint(string(p))))
}

var (
	// ConfigFile specifies the path of the main configuration file relative to BasePath, e.g. "conf/app.conf"
	ConfigFile = path.Join("conf", "app.conf")
	// MimeTypesFile specifies the path of the optional MIME type configuration file relative to BasePath, e.g. "conf/mime-types.conf"
	MimeTypesFile = path.Join("conf", "mime-types.conf")
	// RoutesFile specified the path of the route configuration file relative to BasePath, e.g. "conf/routes"
	RoutesFile = path.Join("conf", "routes")
	// ViewPath specifies the name of directory where all the templates are located relative to BasePath, e.g. "views"
	ViewsPath = "views"

	Config     = NewEmptyConfig()
	MimeConfig = NewEmptyConfig()

	// App details
	AppName  = "(not set)" // e.g. "sample"
	AppRoot  = ""          // e.g. "/app1"
	BasePath = "."         // e.g. "/Users/robfig/gocode/src/corp/sample"

	RunMode = "prod"
	DevMode = false

	// Server config.
	//
	// Alert: This is how the app is configured, which may be different from
	// the current process reality.  For example, if the app is configured for
	// port 9000, HttpPort will always be 9000, even though in dev mode it is
	// run on a random port and proxied.
	HttpPort    = 9000
	HttpAddr    = ""    // e.g. "", "127.0.0.1"
	HttpSsl     = false // e.g. true if using ssl
	HttpSslCert = ""    // e.g. "/path/to/cert.pem"
	HttpSslKey  = ""    // e.g. "/path/to/key.pem"

	// All cookies dropped by the framework begin with this prefix.
	CookiePrefix = "MARS"
	// Cookie domain
	CookieDomain = ""
	// Cookie flags
	CookieHttpOnly = false
	CookieSecure   = false

	// Delimiters to use when rendering templates
	TemplateDelims = ""

	//Logger colors
	colors = map[string]gocolorize.Colorize{
		"trace": gocolorize.NewColor("magenta"),
		"info":  gocolorize.NewColor("green"),
		"warn":  gocolorize.NewColor("yellow"),
		"error": gocolorize.NewColor("red"),
	}

	// Loggers
	DisabledLogger = log.New(ioutil.Discard, "", 0)

	TRACE = DisabledLogger
	INFO  = log.New(&marsLogs{c: colors["info"], w: os.Stderr}, "INFO  ", defaultLoggerFlags)
	WARN  = log.New(&marsLogs{c: colors["warn"], w: os.Stderr}, "WARN  ", defaultLoggerFlags)
	ERROR = log.New(&marsLogs{c: colors["error"], w: os.Stderr}, "ERROR ", defaultLoggerFlags)

	MaxAge = time.Hour * 24 // MaxAge specifies the time browsers shall cache static content served using Static.Serve

	// Private
	secretKey []byte // Key used to sign cookies. An empty key disables signing.
)

func SetAppSecret(secret string) {
	secretKey = []byte(secret)
}

func init() {
	log.SetFlags(defaultLoggerFlags)
}

// InitDefaults initializes Mars based on runtime-loading of config files.
//
// Params:
//   mode - the run mode, which determines which app.conf settings are used.
//   basePath - the path to the configuration, messages, and view directories
func InitDefaults(mode, basePath string) {
	RunMode = mode

	if runtime.GOOS == "windows" {
		gocolorize.SetPlain(true)
	}

	BasePath = filepath.FromSlash(basePath)

	// Load app.conf
	var err error
	Config, err = LoadConfig(path.Join(BasePath, ConfigFile))
	if err != nil || Config == nil {
		log.Fatalln("Failed to load app.conf:", err)
	}

	MimeConfig, _ = LoadConfig(path.Join(BasePath, MimeTypesFile))

	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DEFAULT_SECTION
	}
	if !Config.HasSection(mode) {
		log.Fatalln("app.conf: No mode found:", mode)
	}
	Config.SetSection(mode)

	// Configure properties from app.conf
	DevMode = Config.BoolDefault("mode.dev", DevMode)
	HttpPort = Config.IntDefault("http.port", HttpPort)
	HttpAddr = Config.StringDefault("http.addr", HttpAddr)
	HttpSsl = Config.BoolDefault("http.ssl", HttpSsl)
	HttpSslCert = Config.StringDefault("http.sslcert", HttpSslCert)
	HttpSslKey = Config.StringDefault("http.sslkey", HttpSslKey)

	if HttpSsl {
		if HttpSslCert == "" {
			log.Fatalln("No http.sslcert provided.")
		}
		if HttpSslKey == "" {
			log.Fatalln("No http.sslkey provided.")
		}
	}

	AppName = Config.StringDefault("app.name", AppName)
	AppRoot = Config.StringDefault("app.root", AppRoot)
	CookiePrefix = Config.StringDefault("cookie.prefix", CookiePrefix)
	CookieDomain = Config.StringDefault("cookie.domain", CookieDomain)
	CookieHttpOnly = Config.BoolDefault("cookie.httponly", CookieHttpOnly)
	CookieSecure = Config.BoolDefault("cookie.secure", CookieSecure)
	TemplateDelims = Config.StringDefault("template.delimiters", TemplateDelims)

	if s := Config.StringDefault("app.secret", ""); s != "" {
		SetAppSecret(s)
	}

	// Configure logging
	if !Config.BoolDefault("log.colorize", true) {
		gocolorize.SetPlain(true)
	}

	TRACE = getLogger("trace", TRACE)
	INFO = getLogger("info", INFO)
	WARN = getLogger("warn", WARN)
	ERROR = getLogger("error", ERROR)

	SetupViews()
	SetupRouter()

	INFO.Printf("Initialized Mars v%s (%s) for %s", VERSION, BUILD_DATE, MINIMUM_GO)
}

// SetupViews will create a template loader for all the templates provided in ViewsPath
func SetupViews() {
	MainTemplateLoader = NewTemplateLoader([]string{path.Join(BasePath, ViewsPath)})
	MainTemplateLoader.Refresh()
}

// SetupRouter will create the router of the application based on the information
// provided in RoutesFile and the controllers and actions which have been registered
// using RegisterController.
func SetupRouter() {
	MainRouter = NewRouter(path.Join(BasePath, RoutesFile))
	if err := MainRouter.Refresh(); err != nil {
		ERROR.Fatalln(err.Error())
	}
}

// Create a logger using log.* directives in app.conf plus the current settings
// on the default logger.
func getLogger(name string, original *log.Logger) *log.Logger {
	var logger *log.Logger

	// Create a logger with the requested output. (default to stderr)
	output := Config.StringDefault("log."+name+".output", "")

	switch output {
	case "":
		return original
	case "stdout":
		logger = newLogger(&marsLogs{c: colors[name], w: os.Stdout})
	case "stderr":
		logger = newLogger(&marsLogs{c: colors[name], w: os.Stderr})
	case "off":
		return DisabledLogger
	default:
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file", output, ":", err)
		}
		logger = newLogger(file)
	}

	// Set the prefix / flags.
	flags, found := Config.Int("log." + name + ".flags")
	if found {
		logger.SetFlags(flags)
	}

	prefix, found := Config.String("log." + name + ".prefix")
	if found {
		logger.SetPrefix(prefix)
	}

	return logger
}

func newLogger(wr io.Writer) *log.Logger {
	return log.New(wr, "", defaultLoggerFlags)
}
