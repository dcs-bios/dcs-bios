/*
Package webappserver serves web apps from the file system
or via reverse proxy.

Apps are served under /app/<appname>.
Apps are located on the file system in <appname> folders under the
path that has been passed to the AddHandler function.

If one of these folders contains a proxy.txt file,
requests to that sub-path will ignore the content in that folder
and will instead be reverse-proxied to another HTTP server
that is listed in the proxy.txt file.

proxy.txt is a YAML file that looks like this:

targetUrl: http://hostname:port
stripPrefix: false

The proxy feature can be used during the development of a web app
to be able to serve the app through something like webpack-dev-server
while still allowing the app to access the DCS-BIOS API at a well-known
path and from the same origin.
*/
package webappserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/vulcand/oxy/forward"
	oxyutils "github.com/vulcand/oxy/utils"
)

var settings struct {
	appPath string
}
var staticFileHandler http.Handler
var reverseProxyMap = make(map[string]*httputil.ReverseProxy)

type RewriteRule struct {
	MatchPrefix string `yaml:"matchPrefix"`
	RedirectTo  string `yaml:"redirectTo"`
	StripPrefix string `yaml:"stripPrefix"`
}
type RewriteRules map[string]RewriteRule

type proxyConfig struct {
	TargetURL   string `yaml:"targetUrl"`
	StripPrefix bool   `yaml:"stripPrefix"`
}

func readProxyConfigFile(filename string) (*proxyConfig, error) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := proxyConfig{}
	err = yaml.Unmarshal(fileContents, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func ReadRewriteRules() RewriteRules {
	rules := RewriteRules{}
	fileContents, err := ioutil.ReadFile("rewrite.txt")
	if err != nil {
		log.Println("error reading rewrite rules, returing empty list. Error was:", err.Error())
		return rules
	}

	err = yaml.Unmarshal(fileContents, &rules)
	if err != nil {
		log.Println(err.Error())
	}
	return rules
}

// AddHandler adds a handler for the request path /app/ to the default net.http ServeMux
func AddHandler(appPath string) {
	// settings.appPath is used by the http.FileServer to locate static files
	// and by the requestHandler function to locate the proxy.txt configuration files
	settings.appPath = appPath

	// initialize static file handler to serve static file requests
	staticFileHandler = http.StripPrefix("/app/", http.FileServer(http.Dir(settings.appPath)))

	// add handler
	http.Handle("/app/", http.HandlerFunc(requestHandler))

	log.Print(ReadRewriteRules())
}

// requestHandler dispatches to the static file server or the reverse proxy depending on the existence of a proxy.txt configuration file
func requestHandler(w http.ResponseWriter, r *http.Request) {
	proxyTarget := ""

	fmt.Println("reading rewrite rules")
	// apply rewrite rules
	rewriteRules := ReadRewriteRules()
	for _, rule := range rewriteRules {
		fmt.Println(rule.RedirectTo)
		if strings.HasPrefix(r.RequestURI, rule.MatchPrefix) {
			proxyTarget = rule.RedirectTo
		}
	}

	uriParts := strings.Split(r.RequestURI, "/")
	// uriParts is now ["", "app", <app name>, ...]
	if len(uriParts) < 3 || uriParts[2] == "" { // someone requested /app/
		log.Println("serving URL with static file handler: " + r.RequestURI)
		staticFileHandler.ServeHTTP(w, r)
		return
	}
	// appName := uriParts[2]

	// proxyTxtPath := filepath.Join(settings.appPath, appName, "proxy.txt")
	// proxyConfig, err := readProxyConfigFile(proxyTxtPath)
	// if _, match := err.(*os.PathError); match {
	// 	// no reverse proxy, serve static files
	// 	log.Println("serving URL with static file handler: " + r.RequestURI)
	// 	staticFileHandler.ServeHTTP(w, r)
	// 	return
	// }
	// if err != nil || proxyConfig == nil {
	// 	http.Error(w, "error parsing proxy.txt: "+err.Error(), 500)
	// 	return
	// }

	if proxyTarget == "" {
		// no reverse proxy, serve static files
		log.Println("serving URL with static file handler: " + r.RequestURI)
		staticFileHandler.ServeHTTP(w, r)
		return
	}

	// proxy.txt exists, forward this request to another server
	// targetUrl, err := url.Parse(string(proxyConfig.TargetURL))
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("Invalid proxy configuration string: %s", proxyConfig.TargetURL), 500)
	// 	return
	// }

	handleProxyError := func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "proxy error: "+err.Error(), 500)
		w.Write([]byte(`
The above error occurred while trying to forward the request to another server.
You probably have an error in your proxy.txt configuration file.

Here is an example for a valid proxy.txt file
(the stripPrefix setting is optional):

targetUrl: http://localhost:8000
stripPrefix: false

Here are the contents of your proxy.txt file:

`))
		// contents, _ := ioutil.ReadFile(proxyTxtPath)
		// w.Write(contents)
		return
	}

	// initialize forwarder instance to serve reverse proxy requests
	forwarder, err := forward.New(forward.ErrorHandler(oxyutils.ErrorHandlerFunc(handleProxyError)))
	if err != nil {
		msg := "could not create forwarder instance for reverse proxy: " + err.Error()
		log.Fatalln(msg)
		http.Error(w, msg, 500)
		return
	}

	redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL, err = url.Parse(proxyTarget)
		if err != nil {
			http.Error(w, "could not parse proxy target URL: "+proxyTarget, 500)
			return
		}

		log.Println("serving URL with proxy handler: " + r.RequestURI)
		forwarder.ServeHTTP(w, r)
		return
	})

	prefix := ""
	// if proxyConfig.StripPrefix {
	// 	prefix = strings.Join(uriParts[:3], "/") // "/app/appName"
	// 	log.Println("stripping prefix: " + prefix)
	// }
	http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.StripPrefix strips the prefix from r.URL, we still need to strip it from r.RequestURI
		r.RequestURI = strings.TrimPrefix(r.RequestURI, prefix)
		redirect.ServeHTTP(w, r)
	})).ServeHTTP(w, r)
	return
}

func ProxyRequest(w http.ResponseWriter, r *http.Request, targetURL string, stripPrefix string) {
	handleProxyError := func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "proxy error: "+err.Error(), 500)
		w.Write([]byte(`
The above error occurred while trying to forward the request to another server.
You probably have an error in your redirect.txt configuration file.


`))
	}

	// initialize forwarder instance to serve reverse proxy requests
	forwarder, err := forward.New(forward.ErrorHandler(oxyutils.ErrorHandlerFunc(handleProxyError)))
	if err != nil {
		msg := "could not create forwarder instance for reverse proxy: " + err.Error()
		log.Println(msg)
		http.Error(w, msg, 500)
		return
	}

	redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL, err = url.Parse(targetURL)
		if err != nil {
			http.Error(w, "could not parse proxy target URL: "+targetURL, 500)
			return
		}

		log.Println("serving URL with proxy handler: " + r.RequestURI)
		forwarder.ServeHTTP(w, r)
		return
	})

	http.StripPrefix(stripPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.StripPrefix strips the prefix from r.URL, we still need to strip it from r.RequestURI
		r.RequestURI = strings.TrimPrefix(r.RequestURI, stripPrefix)
		redirect.ServeHTTP(w, r)
	})).ServeHTTP(w, r)
	return
}
