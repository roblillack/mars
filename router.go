package mars

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/roblillack/mars/internal/pathtree"
)

type Route struct {
	Method         string   // e.g. GET
	Path           string   // e.g. /app/:id
	Action         string   // e.g. "Application.ShowApp", "404"
	ControllerName string   // e.g. "Application", ""
	MethodName     string   // e.g. "ShowApp", ""
	FixedParams    []string // e.g. "arg1","arg2","arg3" (CSV formatting)

	routesPath string // e.g. /Users/robfig/gocode/src/myapp/conf/routes
	line       int    // e.g. 3
}

type RouteMatch struct {
	Action         string // e.g. 404
	ControllerName string // e.g. Application
	MethodName     string // e.g. ShowApp
	FixedParams    []string
	Params         map[string][]string // e.g. {id: 123}
}

// Prepares the route to be used in matching.
func NewRoute(method, path, action, fixedArgs, routesPath string, line int) (r *Route) {

	// Handle fixed arguments
	argsReader := strings.NewReader(fixedArgs)
	csv := csv.NewReader(argsReader)
	csv.TrimLeadingSpace = true
	fargs, err := csv.Read()
	if err != nil && err != io.EOF {
		ERROR.Printf("Invalid fixed parameters (%v): for string '%v'", err.Error(), fixedArgs)
	}

	r = &Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		Action:      action,
		FixedParams: fargs,
		routesPath:  routesPath,
		line:        line,
	}

	// URL pattern
	if !strings.HasPrefix(r.Path, "/") {
		ERROR.Print("Absolute URL required.")
		return
	}

	actionSplit := strings.Split(action, ".")
	if len(actionSplit) == 2 {
		r.ControllerName = actionSplit[0]
		r.MethodName = actionSplit[1]
	}

	return
}

func (r Route) TreePath() string {
	method := r.Method
	if method == "*" {
		method = ":METHOD"
	}
	return "/" + method + r.Path
}

type Router struct {
	Routes []*Route
	Tree   *pathtree.Node
	path   string // path to the routes file
}

var notFound = &RouteMatch{Action: "404"}

func (router *Router) Route(req *http.Request) *RouteMatch {
	// Override method if set in header
	if method := req.Header.Get("X-HTTP-Method-Override"); method != "" && req.Method == "POST" {
		req.Method = method
	}

	if router == nil {
		return nil
	}

	leaf, expansions := router.Tree.Find(fmt.Sprintf("/%s%s", req.Method, req.URL.Path))
	if leaf == nil {
		return nil
	}
	route := leaf.Value.(*Route)

	// Create a map of the route parameters.
	var params url.Values
	if len(expansions) > 0 {
		params = make(url.Values)
		for idx, val := range expansions {
			if len(leaf.Wildcards) > idx {
				params[leaf.Wildcards[idx]] = []string{val}
			} else if idx == len(leaf.Wildcards) && leaf.ExtWildcard != "" {
				params[leaf.ExtWildcard] = []string{val}
			}
		}
	}

	// Special handling for explicit 404's.
	if route.Action == "404" {
		return notFound
	}

	// If the action is variablized, replace into it with the captured args.
	controllerName, methodName := route.ControllerName, route.MethodName
	if controllerName[0] == ':' {
		controllerName = params[controllerName[1:]][0]
	}
	if methodName[0] == ':' {
		methodName = params[methodName[1:]][0]
	}

	return &RouteMatch{
		ControllerName: controllerName,
		MethodName:     methodName,
		Params:         params,
		FixedParams:    route.FixedParams,
	}
}

// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() error {
	if r, err := parseRoutesFile(router.path, "", true); err != nil {
		return err
	} else {
		router.Routes = r
	}

	if err := router.updateTree(); err != nil {
		return err
	}

	return nil
}

func (router *Router) updateTree() *Error {
	router.Tree = pathtree.New()

	for _, route := range router.Routes {
		err := router.Tree.Add(route.TreePath(), route)

		// Allow GETs to respond to HEAD requests.
		if err == nil && route.Method == "GET" {
			err = router.Tree.Add("/HEAD"+route.Path, route)
		}

		// Error adding a route to the pathtree.
		if err != nil {
			return routeError(err, route.routesPath, "", route.line)
		}
	}

	return nil
}

// parseRoutesFile reads the given routes file and returns the contained routes.
func parseRoutesFile(routesPath, joinedPath string, validate bool) ([]*Route, *Error) {
	contentBytes, err := ioutil.ReadFile(routesPath)
	if err != nil {
		return nil, &Error{
			Title:       "Failed to load routes file",
			Description: err.Error(),
		}
	}

	return parseRoutes(routesPath, joinedPath, string(contentBytes), validate)
}

// parseRoutes reads the content of a routes file into the routing table.
func parseRoutes(routesFilePath, joinedPath, content string, validate bool) ([]*Route, *Error) {
	var routes []*Route

	// For each line..
	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// A single route
		method, path, action, fixedArgs, found := parseRouteLine(line)
		if !found {
			continue
		}

		// this will avoid accidental double forward slashes in a route.
		// this also avoids pathtree freaking out and causing a runtime panic
		// because of the double slashes
		if strings.HasSuffix(joinedPath, "/") && strings.HasPrefix(path, "/") {
			joinedPath = joinedPath[0 : len(joinedPath)-1]
		}
		path = strings.Join([]string{AppRoot, joinedPath, path}, "")

		route := NewRoute(method, path, action, fixedArgs, routesFilePath, n)
		routes = append(routes, route)

		if validate {
			if err := validateRoute(route); err != nil {
				return nil, routeError(err, routesFilePath, content, n)
			}
		}
	}

	return routes, nil
}

// validateRoute checks that every specified action exists.
func validateRoute(route *Route) error {
	// Skip 404s
	if route.Action == "404" {
		return nil
	}

	// We should be able to load the action.
	parts := strings.Split(route.Action, ".")
	if len(parts) != 2 {
		return fmt.Errorf("Expected two parts (Controller.Action), but got %d: %s",
			len(parts), route.Action)
	}

	// Skip variable routes.
	if parts[0][0] == ':' || parts[1][0] == ':' {
		return nil
	}

	var c Controller
	if err := c.SetAction(parts[0], parts[1]); err != nil {
		return err
	}

	return nil
}

// routeError adds context to a simple error message.
func routeError(err error, routesPath, content string, n int) *Error {
	if marsError, ok := err.(*Error); ok {
		return marsError
	}
	// Load the route file content if necessary
	if content == "" {
		contentBytes, err := ioutil.ReadFile(routesPath)
		if err != nil {
			ERROR.Printf("Failed to read route file %s: %s\n", routesPath, err)
		} else {
			content = string(contentBytes)
		}
	}
	return &Error{
		Title:       "Route validation error",
		Description: err.Error(),
		Path:        routesPath,
		Line:        n + 1,
		SourceLines: strings.Split(content, "\n"),
	}
}

// Groups:
// 1: method
// 4: path
// 5: action
// 6: fixedargs
var routePattern *regexp.Regexp = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|WS|\\*)" +
		"[(]?([^)]*)(\\))?[ \t]+" +
		"(.*/[^ \t]*)[ \t]+([^ \t(]+)" +
		`\(?([^)]*)\)?[ \t]*$`)

func parseRouteLine(line string) (method, path, action, fixedArgs string, found bool) {
	var matches []string = routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action, fixedArgs = matches[1], matches[4], matches[5], matches[6]
	found = true
	return
}

func NewRouter(routesPath string) *Router {
	return &Router{
		Tree: pathtree.New(),
		path: routesPath,
	}
}

type ActionDefinition struct {
	Host, Method, Url, Action string
	Star                      bool
	Args                      map[string]string
}

func (a *ActionDefinition) String() string {
	return a.Url
}

func shouldEscape(c byte) bool {
	if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' {
		return false
	}

	switch c {
	case '-', '_', '.', '~', '$', '&', '+', ':', '=', '@':
		return false
	}

	return true
}

func encodePathSegment(s string) string {
	// From Go 1.8+: return url.PathEscape(segment)
	spaceCount, hexCount := 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			hexCount++
		}
	}

	if spaceCount == 0 && hexCount == 0 {
		return s
	}

	t := make([]byte, len(s)+2*hexCount)
	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case shouldEscape(c):
			t[j] = '%'
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

func (router *Router) Reverse(action string, argValues map[string]string) *ActionDefinition {
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		ERROR.Print("mars/router: reverse router got invalid action ", action)
		return nil
	}
	controllerName, methodName := actionSplit[0], actionSplit[1]

	for _, route := range router.Routes {
		// Skip routes without either a ControllerName or MethodName
		if route.ControllerName == "" || route.MethodName == "" {
			continue
		}

		// Check that the action matches or is a wildcard.
		controllerWildcard := route.ControllerName[0] == ':'
		methodWildcard := route.MethodName[0] == ':'
		if (!controllerWildcard && route.ControllerName != controllerName) ||
			(!methodWildcard && route.MethodName != methodName) {
			continue
		}
		if controllerWildcard {
			argValues[route.ControllerName[1:]] = controllerName
		}
		if methodWildcard {
			argValues[route.MethodName[1:]] = methodName
		}

		// Build up the URL.
		var (
			queryValues  = make(url.Values)
			pathElements = strings.Split(route.Path, "/")
		)
		extension := ""
		for i, el := range pathElements {
			if el == "" || (el[0] != ':' && el[0] != '*') {
				continue
			}

			if dotIdx := strings.IndexRune(el[1:], '.'); dotIdx > 0 {
				extension = el[1+dotIdx:]
				el = el[0 : dotIdx+1]
			}

			val, ok := argValues[el[1:]]
			if !ok {
				val = "<nil>"
				ERROR.Print("mars/router: reverse route missing route arg ", el[1:])
			}
			if el[0] == '*' {
				pathElements[i] = (&url.URL{Path: val}).RequestURI() + extension
			} else {
				pathElements[i] = encodePathSegment(val) + extension
			}
			delete(argValues, el[1:])
			continue
		}

		// Add any args that were not inserted into the path into the query string.
		for k, v := range argValues {
			queryValues.Set(k, v)
		}

		// Calculate the final URL and Method
		url := strings.Join(pathElements, "/")
		if len(queryValues) > 0 {
			url += "?" + queryValues.Encode()
		}

		method := route.Method
		star := false
		if route.Method == "*" {
			method = "GET"
			star = true
		}

		return &ActionDefinition{
			Url:    url,
			Method: method,
			Star:   star,
			Action: action,
			Args:   argValues,
			Host:   "TODO",
		}
	}
	ERROR.Println("Failed to find reverse route:", action, argValues)
	return nil
}

func RouterFilter(c *Controller, fc []Filter) {
	// Figure out the Controller/Action
	var route *RouteMatch = MainRouter.Route(c.Request.Request)
	if route == nil {
		c.Result = c.NotFound("No matching route found: " + c.Request.RequestURI)
		return
	}

	// The route may want to explicitly return a 404.
	if route.Action == "404" {
		c.Result = c.NotFound("(intentionally)")
		return
	}

	// Set the action.
	if err := c.SetAction(route.ControllerName, route.MethodName); err != nil {
		c.Result = c.NotFound(err.Error())
		return
	}

	// Add the route and fixed params to the Request Params.
	c.Params.Route = route.Params

	// Add the fixed parameters mapped by name.
	// TODO: Pre-calculate this mapping.
	for i, value := range route.FixedParams {
		if c.Params.Fixed == nil {
			c.Params.Fixed = make(url.Values)
		}
		if i < len(c.MethodType.Args) {
			arg := c.MethodType.Args[i]
			c.Params.Fixed.Set(arg.Name, value)
		} else {
			WARN.Println("Too many parameters to", route.Action, "trying to add", value)
			break
		}
	}

	fc[0](c, fc[1:])
}

// Override allowed http methods via form or browser param
func HttpMethodOverride(c *Controller, fc []Filter) {
	// An array of HTTP verbs allowed.
	verbs := []string{"POST", "PUT", "PATCH", "DELETE"}

	method := strings.ToUpper(c.Request.Request.Method)

	if method == "POST" {
		param := strings.ToUpper(c.Request.Request.PostFormValue("_method"))

		if len(param) > 0 {
			override := false
			// Check if param is allowed
			for _, verb := range verbs {
				if verb == param {
					override = true
					break
				}
			}

			if override {
				c.Request.Request.Method = param
			} else {
				c.Response.Status = 405
				c.Result = c.RenderError(&Error{
					Title:       "Method not allowed",
					Description: "Method " + param + " is not allowed (valid: " + strings.Join(verbs, ", ") + ")",
				})
				return
			}

		}
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}
