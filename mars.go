package mars

import (
	"fmt"
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
	HttpAddr    = ":9000" // e.g. "", "127.0.0.1"
	HttpSsl     = false   // e.g. true if using ssl
	HttpSslCert = ""      // e.g. "/path/to/cert.pem"
	HttpSslKey  = ""      // e.g. "/path/to/key.pem"

	DualStackHTTP          = false
	SSLAddr                = ":https"
	SelfSignedCert         = false
	SelfSignedOrganization = "ACME Inc."
	SelfSignedDomains      = "127.0.0.1"

	// All cookies dropped by the framework begin with this prefix.
	CookiePrefix = "MARS"
	// Cookie domain
	CookieDomain = ""
	// Cookie flags
	CookieHttpOnly = false
	CookieSecure   = false

	// DisableCSRF disables CSRF checking altogether. See CSRFFilter for more information.
	DisableCSRF = false

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

	setupDone bool
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

	var cfgPath string
	if filepath.IsAbs(ConfigFile) {
		cfgPath = ConfigFile
	} else {
		cfgPath = filepath.Join(BasePath, ConfigFile)
	}

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		var err error
		Config, err = LoadConfig(cfgPath)
		if err != nil || Config == nil {
			log.Fatalln("Failed to load app.conf:", err)
		}
	}

	MimeConfig, _ = LoadConfig(path.Join(BasePath, MimeTypesFile))

	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DEFAULT_SECTION
	}
	if Config.HasSection(mode) {
		Config.SetSection(mode)
	}

	// Configure properties from app.conf
	DevMode = Config.BoolDefault("mode.dev", DevMode)
	HttpAddr = Config.StringDefault("http.addr", HttpAddr)
	HttpSsl = Config.BoolDefault("https.enabled", Config.BoolDefault("http.ssl", HttpSsl))
	HttpSslCert = Config.StringDefault("https.certfile", Config.StringDefault("http.sslcert", HttpSslCert))
	HttpSslKey = Config.StringDefault("https.keyfile", Config.StringDefault("http.sslkey", HttpSslKey))

	DualStackHTTP = Config.BoolDefault("http.dualstack", DualStackHTTP)
	SSLAddr = Config.StringDefault("https.addr", "")
	SelfSignedCert = Config.BoolDefault("https.selfsign", SelfSignedCert)
	SelfSignedOrganization = Config.StringDefault("https.organization", SelfSignedOrganization)
	SelfSignedDomains = Config.StringDefault("https.domains", SelfSignedDomains)

	if (DualStackHTTP || HttpSsl) && !SelfSignedCert {
		if HttpSslCert == "" {
			log.Fatalln("No https.certfile provided and https.selfsign not true.")
		}
		if HttpSslKey == "" {
			log.Fatalln("No https.keyfile provided and https.selfsign not true.")
		}
	}

	tryAddingSSLPort := false
	// Support legacy way of specifying HTTPS addr
	if SSLAddr == "" {
		if HttpSsl && !DualStackHTTP {
			SSLAddr = HttpAddr
			tryAddingSSLPort = true
		} else {
			SSLAddr = ":https"
		}
	}

	// Support legacy way of specifying port number as config setting http.port
	if p := Config.IntDefault("http.port", -1); p != -1 {
		HttpAddr = fmt.Sprintf("%s:%d", HttpAddr, p)
		if tryAddingSSLPort {
			SSLAddr = fmt.Sprintf("%s:%d", SSLAddr, p)
		}
	}

	AppName = Config.StringDefault("app.name", AppName)
	AppRoot = Config.StringDefault("app.root", AppRoot)
	CookiePrefix = Config.StringDefault("cookie.prefix", CookiePrefix)
	CookieDomain = Config.StringDefault("cookie.domain", CookieDomain)
	CookieHttpOnly = Config.BoolDefault("cookie.httponly", CookieHttpOnly)
	CookieSecure = Config.BoolDefault("cookie.secure", CookieSecure)

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

	setup()
}

func setup() {
	// The "watch" config variable can turn on and off all watching.
	// (As a convenient way to control it all together.)
	if Config.BoolDefault("watch", DevMode) {
		MainWatcher = NewWatcher()
		Filters = append([]Filter{WatchFilter}, Filters...)
	}

	if MainTemplateLoader == nil {
		SetupViews()
	}
	if MainRouter == nil {
		SetupRouter()
	}

	runStartupHooks()

	setupDone = true
}

// initializeFallbacks will setup all configuration options that are needed for serving results but might not have
// been initialized correctly by the consumer of the toolkit
func initializeFallbacks() {
	if MainTemplateLoader == nil {
		MainTemplateLoader = emptyTemplateLoader()
	}
}

// SetupViews will create a template loader for all the templates provided in ViewsPath
func SetupViews() {
	MainTemplateLoader = NewTemplateLoader([]string{path.Join(BasePath, ViewsPath)})
	if err := MainTemplateLoader.Refresh(); err != nil {
		ERROR.Fatalln(err.Error())
	}

	// If desired (or by default), create a watcher for templates and routes.
	// The watcher calls Refresh() on things on the first request.
	if MainWatcher != nil && Config.BoolDefault("watch.templates", true) {
		MainWatcher.Listen(MainTemplateLoader, MainTemplateLoader.paths...)
	}
}

// SetupRouter will create the router of the application based on the information
// provided in RoutesFile and the controllers and actions which have been registered
// using RegisterController.
func SetupRouter() {
	MainRouter = NewRouter(filepath.Join(BasePath, RoutesFile))
	if err := MainRouter.Refresh(); err != nil {
		ERROR.Fatalln(err.Error())
	}

	// If desired (or by default), create a watcher for templates and routes.
	// The watcher calls Refresh() on things on the first request.
	if MainWatcher != nil && Config.BoolDefault("watch.routes", true) {
		MainWatcher.Listen(MainRouter, MainRouter.path)
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
