// This package will be shared between Revel and Revel CLI eventually
package model

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/revel/cmd/utils"
	"github.com/revel/config"
	"golang.org/x/tools/go/packages"
)

// Error is used for constant errors.
type Error string

// Error implements the error interface.
func (e Error) Error() string {
	return string(e)
}

const (
	ErrNoApp       Error = "no app found at path"
	ErrNoConfig    Error = "no config found at path"
	ErrNotFound    Error = "not found"
	ErrMissingCert Error = "no http.sslcert provided"
	ErrMissingKey  Error = "no http.sslkey provided"
	ErrNoFiles     Error = "no files found in import path"
	ErrNoPackages  Error = "no packages found for import"
)

type (
	// The container object for describing all Revels variables.
	RevelContainer struct {
		BuildPaths struct {
			Revel string
		}
		Paths struct {
			Import   string
			Source   string
			Base     string
			App      string
			Views    string
			Code     []string
			Template []string
			Config   []string
		}
		PackageInfo struct {
			Config   config.Context
			Packaged bool
			DevMode  bool
			Vendor   bool
		}
		Application struct {
			Name string
			Root string
		}

		ImportPath    string                 // The import path
		SourcePath    string                 // The full source path
		RunMode       string                 // The current run mode
		RevelPath     string                 // The path to the Revel source code
		BasePath      string                 // The base path to the application
		AppPath       string                 // The application path (BasePath + "/app")
		ViewsPath     string                 // The application views path
		CodePaths     []string               // All the code paths
		TemplatePaths []string               // All the template paths
		ConfPaths     []string               // All the configuration paths
		Config        *config.Context        // The global config object
		Packaged      bool                   // True if packaged
		DevMode       bool                   // True if running in dev mode
		HTTPPort      int                    // The http port
		HTTPAddr      string                 // The http address
		HTTPSsl       bool                   // True if running https
		HTTPSslCert   string                 // The SSL certificate
		HTTPSslKey    string                 // The SSL key
		AppName       string                 // The application name
		AppRoot       string                 // The application root from the config `app.root`
		CookiePrefix  string                 // The cookie prefix
		CookieDomain  string                 // The cookie domain
		CookieSecure  bool                   // True if cookie is secure
		SecretStr     string                 // The secret string
		MimeConfig    *config.Context        // The mime configuration
		ModulePathMap map[string]*ModuleInfo // The module path map
	}
	ModuleInfo struct {
		ImportPath string
		Path       string
	}

	WrappedRevelCallback struct {
		FireEventFunction func(key Event, value interface{}) (response EventResponse)
		ImportFunction    func(pkgName string) error
	}
)

// Simple Wrapped RevelCallback.
func NewWrappedRevelCallback(fe func(key Event, value interface{}) (response EventResponse), ie func(pkgName string) error) RevelCallback {
	return &WrappedRevelCallback{fe, ie}
}

// Function to implement the FireEvent.
func (w *WrappedRevelCallback) FireEvent(key Event, value interface{}) (response EventResponse) {
	if w.FireEventFunction != nil {
		response = w.FireEventFunction(key, value)
	}
	return
}

func (w *WrappedRevelCallback) PackageResolver(pkgName string) error {
	return w.ImportFunction(pkgName)
}

// RevelImportPath Revel framework import path.
var (
	RevelImportPath        = "github.com/revel/revel"
	RevelModulesImportPath = "github.com/revel/modules"
)

// This function returns a container object describing the revel application
// eventually this type of function will replace the global variables.
func NewRevelPaths(mode, importPath, appSrcPath string, callback RevelCallback) (rp *RevelContainer, err error) {
	rp = &RevelContainer{ModulePathMap: map[string]*ModuleInfo{}}
	// Ignore trailing slashes.
	rp.ImportPath = strings.TrimRight(importPath, "/")
	rp.SourcePath = appSrcPath
	rp.RunMode = mode

	// We always need to determine the paths for files
	pathMap, err := utils.FindSrcPaths(appSrcPath, []string{importPath + "/app", RevelImportPath}, callback.PackageResolver)
	if err != nil {
		return
	}
	rp.AppPath, rp.RevelPath = pathMap[importPath], pathMap[RevelImportPath]
	// Setup paths for application
	rp.BasePath = rp.SourcePath
	rp.PackageInfo.Vendor = utils.Exists(filepath.Join(rp.BasePath, "go.mod"))
	rp.AppPath = filepath.Join(rp.BasePath, "app")

	// Sanity check , ensure app and conf paths exist
	if !utils.DirExists(rp.AppPath) {
		return rp, fmt.Errorf("%w: %s", ErrNoApp, rp.AppPath)
	}
	if !utils.DirExists(filepath.Join(rp.BasePath, "conf")) {
		return rp, fmt.Errorf("%w: %s", ErrNoConfig, filepath.Join(rp.BasePath, "conf"))
	}

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

	rp.Config, err = config.LoadContext("app.conf", rp.ConfPaths)
	if err != nil {
		return rp, fmt.Errorf("unable to load configuration file %w", err)
	}

	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DefaultSection
	}
	if !rp.Config.HasSection(mode) {
		return rp, fmt.Errorf("app.conf: %w %s %s", ErrNotFound, "run-mode", mode)
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
			return rp, ErrMissingCert
		}

		if rp.HTTPSslKey == "" {
			return rp, ErrMissingKey
		}
	}

	rp.AppName = rp.Config.StringDefault("app.name", "(not set)")
	rp.AppRoot = rp.Config.StringDefault("app.root", "")
	rp.CookiePrefix = rp.Config.StringDefault("cookie.prefix", "REVEL")
	rp.CookieDomain = rp.Config.StringDefault("cookie.domain", "")
	rp.CookieSecure = rp.Config.BoolDefault("cookie.secure", rp.HTTPSsl)
	rp.SecretStr = rp.Config.StringDefault("app.secret", "")

	callback.FireEvent(REVEL_BEFORE_MODULES_LOADED, nil)
	utils.Logger.Info("Loading modules")
	if err := rp.loadModules(callback); err != nil {
		return rp, err
	}

	callback.FireEvent(REVEL_AFTER_MODULES_LOADED, nil)

	return
}

// LoadMimeConfig load mime-types.conf on init.
func (rp *RevelContainer) LoadMimeConfig() (err error) {
	rp.MimeConfig, err = config.LoadContext("mime-types.conf", rp.ConfPaths)
	if err != nil {
		return fmt.Errorf("failed to load mime type config: %s %w", "error", err)
	}
	return
}

// Loads modules based on the configuration setup.
// This will fire the REVEL_BEFORE_MODULE_LOADED, REVEL_AFTER_MODULE_LOADED
// for each module loaded. The callback will receive the RevelContainer, name, moduleImportPath and modulePath
// It will automatically add in the code paths for the module to the
// container object.
func (rp *RevelContainer) loadModules(callback RevelCallback) (err error) {
	keys := []string{}
	keys = append(keys, rp.Config.Options("module.")...)

	// Reorder module order by key name, a poor mans sort but at least it is consistent
	sort.Strings(keys)
	for _, key := range keys {
		moduleImportPath := rp.Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := rp.ResolveImportPath(moduleImportPath)
		if err != nil {
			utils.Logger.Info("Missing module ", "module_import_path", moduleImportPath, "error", err)

			if err := callback.PackageResolver(moduleImportPath); err != nil {
				return fmt.Errorf("failed to resolve package %w", err)
			}

			modulePath, err = rp.ResolveImportPath(moduleImportPath)
			if err != nil {
				return fmt.Errorf("failed to load module.  Import of path failed %s:%s %s:%w ", "modulePath", moduleImportPath, "error", err)
			}
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
	return
}

// Adds a module paths to the container object.
func (rp *RevelContainer) addModulePaths(name, importPath, modulePath string) {
	utils.Logger.Info("Adding module path", "name", name, "import path", importPath, "system path", modulePath)
	if codePath := filepath.Join(modulePath, "app"); utils.DirExists(codePath) {
		rp.CodePaths = append(rp.CodePaths, codePath)
		rp.ModulePathMap[name] = &ModuleInfo{importPath, modulePath}
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
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir: rp.AppPath,
	}
	config.Env = utils.ReducedEnv(false)
	pkgs, err := packages.Load(config, importPath)
	if len(pkgs) == 0 {
		return "", fmt.Errorf("%w %s using app path %s", ErrNoPackages, importPath, rp.AppPath)
	}
	//	modPkg, err := build.Import(importPath, rp.AppPath, build.FindOnly)
	if err != nil {
		return "", err
	}
	if len(pkgs[0].GoFiles) > 0 {
		return filepath.Dir(pkgs[0].GoFiles[0]), nil
	}
	return pkgs[0].PkgPath, fmt.Errorf("%w: %s", ErrNoFiles, importPath)
}
