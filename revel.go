package mars

import (
	"github.com/agtorre/gocolorize"
	"github.com/robfig/config"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

const (
	MarsImportPath = "github.com/roblillack/mars"
)

type marsLogs struct {
	c gocolorize.Colorize
	w io.Writer
}

func (r *marsLogs) Write(p []byte) (n int, err error) {
	return r.w.Write([]byte(r.c.Paint(string(p))))
}

var (
	// App details
	AppName    string // e.g. "sample"
	AppRoot    string // e.g. "/app1"
	BasePath   string // e.g. "/Users/robfig/gocode/src/corp/sample"
	ImportPath string // e.g. "corp/sample"

	Config     *MergedConfig
	MimeConfig *MergedConfig
	RunMode    string // Application-defined (by default, "dev" or "prod")
	DevMode    bool   // if true, RunMode is a development mode.

	// Where to look for templates and configuration.
	// Ordered by priority.  (Earlier paths take precedence over later paths.)
	CodePaths []string
	ConfPaths []string

	// Server config.
	//
	// Alert: This is how the app is configured, which may be different from
	// the current process reality.  For example, if the app is configured for
	// port 9000, HttpPort will always be 9000, even though in dev mode it is
	// run on a random port and proxied.
	HttpPort    int    // e.g. 9000
	HttpAddr    string // e.g. "", "127.0.0.1"
	HttpSsl     bool   // e.g. true if using ssl
	HttpSslCert string // e.g. "/path/to/cert.pem"
	HttpSslKey  string // e.g. "/path/to/key.pem"

	// All cookies dropped by the framework begin with this prefix.
	CookiePrefix string
	// Cookie domain
	CookieDomain string
	// Cookie flags
	CookieHttpOnly bool
	CookieSecure   bool

	// Delimiters to use when rendering templates
	TemplateDelims string

	//Logger colors
	colors = map[string]gocolorize.Colorize{
		"trace": gocolorize.NewColor("magenta"),
		"info":  gocolorize.NewColor("green"),
		"warn":  gocolorize.NewColor("yellow"),
		"error": gocolorize.NewColor("red"),
	}

	error_log = marsLogs{c: colors["error"], w: os.Stderr}

	// Loggers
	TRACE = log.New(ioutil.Discard, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile)
	INFO  = log.New(ioutil.Discard, "INFO  ", log.Ldate|log.Ltime|log.Lshortfile)
	WARN  = log.New(ioutil.Discard, "WARN  ", log.Ldate|log.Ltime|log.Lshortfile)
	ERROR = log.New(&error_log, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)

	MaxAge = time.Hour * 24 // MaxAge specifies the time browsers shall cache static content served using Static.Serve

	Initialized bool

	// Private
	secretKey []byte // Key used to sign cookies. An empty key disables signing.
)

func init() {
	log.SetFlags(INFO.Flags())
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
	ConfPaths = []string{path.Join(BasePath, "conf")}

	/*
		fmt.Println("RunMode", RunMode)
		fmt.Println("BasePath", BasePath)
	*/

	// Load app.conf
	var err error
	Config, err = LoadConfig(path.Join(BasePath, "conf", "app.conf"))
	if err != nil || Config == nil {
		log.Fatalln("Failed to load app.conf:", err)
	}

	MimeConfig, _ = LoadConfig(path.Join(BasePath, "conf", "mime-types.conf"))

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
	DevMode = Config.BoolDefault("mode.dev", false)
	HttpPort = Config.IntDefault("http.port", 9000)
	HttpAddr = Config.StringDefault("http.addr", "")
	HttpSsl = Config.BoolDefault("http.ssl", false)
	HttpSslCert = Config.StringDefault("http.sslcert", "")
	HttpSslKey = Config.StringDefault("http.sslkey", "")
	if HttpSsl {
		if HttpSslCert == "" {
			log.Fatalln("No http.sslcert provided.")
		}
		if HttpSslKey == "" {
			log.Fatalln("No http.sslkey provided.")
		}
	}

	AppName = Config.StringDefault("app.name", "(not set)")
	AppRoot = Config.StringDefault("app.root", "")
	CookiePrefix = Config.StringDefault("cookie.prefix", "MARS")
	CookieDomain = Config.StringDefault("cookie.domain", "")
	CookieHttpOnly = Config.BoolDefault("cookie.httponly", false)
	CookieSecure = Config.BoolDefault("cookie.secure", false)
	TemplateDelims = Config.StringDefault("template.delimiters", "")
	if secretStr := Config.StringDefault("app.secret", ""); secretStr != "" {
		secretKey = []byte(secretStr)
	}

	// Configure logging
	if !Config.BoolDefault("log.colorize", true) {
		gocolorize.SetPlain(true)
	}

	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")

	MainTemplateLoader = NewTemplateLoader([]string{path.Join(BasePath, "views")})
	MainTemplateLoader.Refresh()

	Initialized = true
	INFO.Printf("Initialized Mars v%s (%s) for %s", VERSION, BUILD_DATE, MINIMUM_GO)
}

// Create a logger using log.* directives in app.conf plus the current settings
// on the default logger.
func getLogger(name string) *log.Logger {
	var logger *log.Logger

	// Create a logger with the requested output. (default to stderr)
	output := Config.StringDefault("log."+name+".output", "stderr")
	var newlog marsLogs

	switch output {
	case "stdout":
		newlog = marsLogs{c: colors[name], w: os.Stdout}
		logger = newLogger(&newlog)
	case "stderr":
		newlog = marsLogs{c: colors[name], w: os.Stderr}
		logger = newLogger(&newlog)
	default:
		if output == "off" {
			output = os.DevNull
		}

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
	return log.New(wr, "", INFO.Flags())
}

// ResolveImportPath returns the filesystem path for the given import path.
// Returns an error if the import path could not be found.
func ResolveImportPath(importPath string) (string, error) {
	// GO15VENDOREXPERIMENT
	var err error
	var modPkg *build.Package
	for _, p := range []string{
		importPath,
		path.Join(ImportPath, "vendor", importPath),
		path.Join(MarsImportPath, "vendor", importPath)} {
		modPkg, err = build.Import(p, "", build.FindOnly)
		if err == nil {
			break
		}
	}

	if err != nil {
		return "", err
	}

	return modPkg.Dir, nil
}

func CheckInit() {
	if !Initialized {
		panic("Mars has not been initialized!")
	}
}
