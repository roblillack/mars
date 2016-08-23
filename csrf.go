package mars

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

const csrfCookieKey = "_csrf"
const csrfCookieName = "CSRF"
const csrfHeaderName = "X-CSRF-Token"
const csrfFieldName = "_csrf_token"

func isSafeMethod(c *Controller) bool {
	// Methods deemed safe as as defined RFC 7231, section 4.2.1.
	// TODO: We might think about adding the two other idempotent methods here, too.
	for _, i := range []string{"GET", "HEAD", "OPTIONS", "TRACE"} {
		if c.Request.Method == i {
			return true
		}

	}

	return false
}

func findCSRFToken(c *Controller) string {
	if h := c.Request.Header.Get(csrfHeaderName); h != "" {
		INFO.Printf("Have header CSRF token: %s\n", h)
		return h
	}

	if f := c.Params.Get(csrfFieldName); f != "" {
		INFO.Printf("Have form field CSRF token: %s\n", f)
		return f
	}

	return ""
}

func generateRandomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		ERROR.Printf("Error generating random CSRF token: %s\n", err)
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(buf)
}

// CSRFFilter provides measures of protecting against attacks known as
// "Cross-site request forgery" multiple ways in which the frontend of
// the application can prove that a mutating request to the server was
// actually initiated by the said frontend and not an attacker, that
// lured the user into calling unwanted on your site.
//
// A random CSRF token is added to the signed session (as key `_csrf`)
// and an additional Cookie (which can be read using JavaScript) called
// `XXX_CSRF`. The token is also available to the template engine as
// `{{.csrfToken}}` or as ready-made, hidden form field using
// `{{.csrfField}}`.
//
// For each HTTP request not deemed safe according to RFC 7231,
// section 4.2.1, one of these methods MUST be used for the server to
// ascertain that the user actually aksed to call this action in the
// first place:
//
// a) The token is sent using a custom header `X-CSRF-Token` with the
// request. This is very useful for single page application and AJAX
// requests, as most frontend toolkits can be set up to include this
// header if needed. An example for jQuery (added to the footer of each
// page) could look like this:
//
//    <script type="text/javascript">
//    $(function() {
//        $.ajaxSetup({
//            crossDomain: false,
//            beforeSend: function(xhr, settings) {
//                // HTTP methods that do not require CSRF protection.
//                if (!/^(GET|HEAD|OPTIONS|TRACE)$/.test(settings.type)) {
//                    xhr.setRequestHeader("X-CSRF-Token", {{.csrfToken}});
//                }
//            }
//        });
//    });
//    </script>
//
// b) The token is sent as a form field value for forms using non-safe
// actions. Simply adding `{{.csrfField}}`` should be enough.
//
// To disable CSRF protection for individual actions or controllers
// (ie. API calls that authenticate using HTTP Basic Auth or AccessTokens,
// etc.), add an InterceptorMethod to your Controller that sets the
// Controller.DisableCSRF to `true` for said requests.
//
// See also:
// https://tools.ietf.org/html/rfc7231#section-4.2.1
var CSRFFilter = func(c *Controller, fc []Filter) {
	if DisableCSRF {
		fc[0](c, fc[1:])
		return
	}

	csrfToken := c.Session[csrfCookieKey]
	if len(csrfToken) != 22 {
		csrfToken = generateRandomToken()
		c.Session[csrfCookieKey] = csrfToken
		TRACE.Printf("Created session token: %s\n", csrfToken)
	} else {
		TRACE.Printf("Browser sent session token: %s\n", csrfToken)
	}

	c.SetCookie(&http.Cookie{
		Name:     fmt.Sprintf("%s_%s", CookiePrefix, csrfCookieName),
		Value:    csrfToken,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: false,
		Secure:   CookieSecure,
		Expires:  time.Now().Add(12 * time.Hour).UTC(),
	})
	c.RenderArgs["csrfToken"] = csrfToken
	c.RenderArgs["csrfField"] = template.HTML(`<input type='hidden' name='` + csrfFieldName + `' value='` + csrfToken + `'/>`)

	ignore, haveIgnore := c.Args["ignore_csrf"].(bool)
	if !isSafeMethod(c) && !(haveIgnore && ignore) {
		token := findCSRFToken(c)
		if token == "" || token != csrfToken {
			c.Result = c.Forbidden("No/wrong CSRF token given.")
			return
		}
	}

	fc[0](c, fc[1:])
}
