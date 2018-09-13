// This package will be shared between Revel and Revel CLI eventually
package model

import (
	"github.com/revel/cmd/utils"
	"github.com/revel/config"
	"go/build"

	"os"
	"path/filepath"
	"sort"
	"strings"
)
const (
	// Event type when templates are going to be refreshed (receivers are registered template engines added to the template.engine conf option)
	TEMPLATE_REFRESH_REQUESTED = iota
	// Event type when templates are refreshed (receivers are registered template engines added to the template.engine conf option)
	TEMPLATE_REFRESH_COMPLETED
	// Event type before all module loads, events thrown to handlers added to AddInitEventHandler

	// Event type before all module loads, events thrown to handlers added to AddInitEventHandler
	REVEL_BEFORE_MODULES_LOADED
	// Event type called when a new module is found
	REVEL_BEFORE_MODULE_LOADED
	// Event type called when after a new module is found
	REVEL_AFTER_MODULE_LOADED
	// Event type after all module loads, events thrown to handlers added to AddInitEventHandler
	REVEL_AFTER_MODULES_LOADED

	// Event type before server engine is initialized, receivers are active server engine and handlers added to AddInitEventHandler
	ENGINE_BEFORE_INITIALIZED
	// Event type before server engine is started, receivers are active server engine and handlers added to AddInitEventHandler
	ENGINE_STARTED
	// Event type after server engine is stopped, receivers are active server engine and handlers added to AddInitEventHandler
	ENGINE_SHUTDOWN

	// Called before routes are refreshed
	ROUTE_REFRESH_REQUESTED
	// Called after routes have been refreshed
	ROUTE_REFRESH_COMPLETED
)
type (
	// The container object for describing all Revels variables
	RevelContainer struct {
		ImportPath    string // The import path
		SourcePath    string // The full source path
		RunMode       string // The current run mode
		RevelPath     string // The path to the Revel source code
		BasePath      string // The base path to the application
		AppPath       string // The application path (BasePath + "/app"
		ViewsPath     string // The application views path
		CodePaths     []string // All the code paths
		TemplatePaths []string // All the template paths
		ConfPaths     []string // All the configuration paths
		Config        *config.Context // The global config object
		Packaged      bool // True if packaged
		DevMode       bool // True if running in dev mode
		HTTPPort      int // The http port
		HTTPAddr      string // The http address
		HTTPSsl       bool // True if running https
		HTTPSslCert   string // The SSL certificate
		HTTPSslKey    string // The SSL key
		AppName       string // The application name
		AppRoot       string // The application root from the config `app.root`
		CookiePrefix  string // The cookie prefix
		CookieDomain  string // The cookie domain
		CookieSecure  bool // True if cookie is secure
		SecretStr     string // The secret string
		MimeConfig    *config.Context // The mime configuration
		ModulePathMap map[string]string // The module path map
	}

	RevelCallback interface {
		FireEvent(key int, value interface{}) (response int)
	}
	doNothingRevelCallback struct {

	}

)

// Simple callback to pass to the RevelCallback that does nothing
var DoNothingRevelCallback = RevelCallback(&doNothingRevelCallback{})

func (_ *doNothingRevelCallback) FireEvent(key int, value interface{}) (response int) {
	return
}

// RevelImportPath Revel framework import path
var RevelImportPath = "github.com/revel/revel"

// This function returns a container object describing the revel application
// eventually this type of function will replace the global variables.
func NewRevelPaths(mode, importPath, srcPath string, callback RevelCallback) (rp *RevelContainer) {
	rp = &RevelContainer{ModulePathMap: map[string]string{}}
	// Ignore trailing slashes.
	rp.ImportPath = strings.TrimRight(importPath, "/")
	rp.SourcePath = srcPath
	rp.RunMode = mode

	// If the SourcePath is not specified, find it using build.Import.
	var revelSourcePath string // may be different from the app source path
	if rp.SourcePath == "" {
		revelSourcePath, rp.SourcePath = findSrcPaths(importPath)
	} else {
		// If the SourcePath was specified, assume both Revel and the app are within it.
		rp.SourcePath = filepath.Clean(rp.SourcePath)
		revelSourcePath = rp.SourcePath

	}

	// Setup paths for application
	rp.RevelPath = filepath.Join(revelSourcePath, filepath.FromSlash(RevelImportPath))
	rp.BasePath = filepath.Join(rp.SourcePath, filepath.FromSlash(importPath))
	rp.AppPath = filepath.Join(rp.BasePath, "app")
	rp.ViewsPath = filepath.Join(rp.AppPath, "views")

	rp.CodePaths = []string{rp.AppPath}
	rp.TemplatePaths = []string{}

	if rp.ConfPaths == nil {
		rp.ConfPaths = []string{}
	}

	// Config load order
	// 1. framework (revel/conf/*)
	// 2. application (conf/*)
	// 3. user supplied configs (...) - User configs can override/add any from above
	rp.ConfPaths = append(
		[]string{
			filepath.Join(rp.RevelPath, "conf"),
			filepath.Join(rp.BasePath, "conf"),
		},
		rp.ConfPaths...)

	var err error
	rp.Config, err = config.LoadContext("app.conf", rp.ConfPaths)
	if err != nil {
		utils.Logger.Fatal("Unable to load configuartion file ","error", err)
		os.Exit(1)
	}

	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DefaultSection
	}
	if !rp.Config.HasSection(mode) {
		utils.Logger.Fatal("app.conf: No mode found:", mode)
	}
	rp.Config.SetSection(mode)

	// Configure properties from app.conf
	rp.DevMode = rp.Config.BoolDefault("mode.dev", false)
	rp.HTTPPort = rp.Config.IntDefault("http.port", 9000)
	rp.HTTPAddr = rp.Config.StringDefault("http.addr", "")
	rp.HTTPSsl = rp.Config.BoolDefault("http.ssl", false)
	rp.HTTPSslCert = rp.Config.StringDefault("http.sslcert", "")
	rp.HTTPSslKey = rp.Config.StringDefault("http.sslkey", "")
	if rp.HTTPSsl {
		if rp.HTTPSslCert == "" {
			utils.Logger.Fatal("No http.sslcert provided.")
		}
		if rp.HTTPSslKey == "" {
			utils.Logger.Fatal("No http.sslkey provided.")
		}
	}
	//
	rp.AppName = rp.Config.StringDefault("app.name", "(not set)")
	rp.AppRoot = rp.Config.StringDefault("app.root", "")
	rp.CookiePrefix = rp.Config.StringDefault("cookie.prefix", "REVEL")
	rp.CookieDomain = rp.Config.StringDefault("cookie.domain", "")
	rp.CookieSecure = rp.Config.BoolDefault("cookie.secure", rp.HTTPSsl)
	rp.SecretStr = rp.Config.StringDefault("app.secret", "")


	callback.FireEvent(REVEL_BEFORE_MODULES_LOADED, nil)
	rp.loadModules(callback)
	callback.FireEvent(REVEL_AFTER_MODULES_LOADED, nil)

	return
}

// LoadMimeConfig load mime-types.conf on init.
func (rp *RevelContainer) LoadMimeConfig() {
	var err error
	rp.MimeConfig, err = config.LoadContext("mime-types.conf", rp.ConfPaths)
	if err != nil {
		utils.Logger.Fatal("Failed to load mime type config:", "error", err)
	}
}

// Loads modules based on the configuration setup.
// This will fire the REVEL_BEFORE_MODULE_LOADED, REVEL_AFTER_MODULE_LOADED
// for each module loaded. The callback will receive the RevelContainer, name, moduleImportPath and modulePath
// It will automatically add in the code paths for the module to the
// container object
func (rp *RevelContainer) loadModules(callback RevelCallback) {
	keys := []string{}
	for _, key := range rp.Config.Options("module.") {
		keys = append(keys, key)
	}

	// Reorder module order by key name, a poor mans sort but at least it is consistent
	sort.Strings(keys)
	for _, key := range keys {
		moduleImportPath := rp.Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := rp.ResolveImportPath(moduleImportPath)
		if err != nil {
			utils.Logger.Error("Failed to load module.  Import of path failed", "modulePath", moduleImportPath, "error", err)
		}
		// Drop anything between module.???.<name of module>
		name := key[len("module."):]
		if index := strings.Index(name, "."); index > -1 {
			name = name[index+1:]
		}
		callback.FireEvent(REVEL_BEFORE_MODULE_LOADED, []interface{}{rp, name, moduleImportPath, modulePath})
		rp.addModulePaths(name, moduleImportPath, modulePath)
		callback.FireEvent(REVEL_AFTER_MODULE_LOADED, []interface{}{rp, name, moduleImportPath, modulePath})
	}
}

// Adds a module paths to the container object
func (rp *RevelContainer) addModulePaths(name, importPath, modulePath string) {
	if codePath := filepath.Join(modulePath, "app"); utils.DirExists(codePath) {
		rp.CodePaths = append(rp.CodePaths, codePath)
		rp.ModulePathMap[name] = modulePath
		if viewsPath := filepath.Join(modulePath, "app", "views"); utils.DirExists(viewsPath) {
			rp.TemplatePaths = append(rp.TemplatePaths, viewsPath)
		}
	}

	// Hack: There is presently no way for the testrunner module to add the
	// "test" subdirectory to the CodePaths.  So this does it instead.
	if importPath == rp.Config.StringDefault("module.testrunner", "github.com/revel/modules/testrunner") {
		joinedPath := filepath.Join(rp.BasePath, "tests")
		rp.CodePaths = append(rp.CodePaths, joinedPath)
	}
	if testsPath := filepath.Join(modulePath, "tests"); utils.DirExists(testsPath) {
		rp.CodePaths = append(rp.CodePaths, testsPath)
	}
}

// ResolveImportPath returns the filesystem path for the given import path.
// Returns an error if the import path could not be found.
func (rp *RevelContainer) ResolveImportPath(importPath string) (string, error) {
	if rp.Packaged {
		return filepath.Join(rp.SourcePath, importPath), nil
	}

	modPkg, err := build.Import(importPath, rp.RevelPath, build.FindOnly)
	if err != nil {
		return "", err
	}
	return modPkg.Dir, nil
}

// Find the full source dir for the import path, uses the build.Default.GOPATH to search for the directory
func findSrcPaths(importPath string) (revelSourcePath, appSourcePath string) {
	var (
		gopaths = filepath.SplitList(build.Default.GOPATH)
		goroot  = build.Default.GOROOT
	)

	if len(gopaths) == 0 {
		utils.Logger.Fatalf("GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	if utils.ContainsString(gopaths, goroot) {
		utils.Logger.Fatalf("GOPATH (%s) must not include your GOROOT (%s). "+
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.",
			gopaths, goroot)

	}

	appPkg, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		utils.Logger.Fatal("Failed to import "+importPath+" with error:", "error", err)
	}

	revelPkg, err := build.Import(RevelImportPath, appPkg.Dir, build.FindOnly)
	if err != nil {
		utils.Logger.Fatal("Failed to find Revel with error:", "error", err)
	}

	revelSourcePath, appSourcePath = revelPkg.Dir[:len(revelPkg.Dir)-len(RevelImportPath)], appPkg.SrcRoot
	return
}

