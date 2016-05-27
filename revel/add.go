package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/revel/revel"
)

var cmdAdd = &Command{
	UsageLine: "add [path] [controller_name]",
	Short:     "add a new controller to the Revel application",
	Long: `
Add creates a new controller file and puts it in the Revel application directory you specify and sets up the controller file appropriately. Also adds the views
directory for the new controller.

For example:
    revel add import/path/app user

This creates a user.go file in the /app/controllers directory of import/path/app and a User directory under /app/views
`,
}

func init() {
	cmdAdd.Run = addController
}

var (
	// revel related paths
	revelPath  string
	controller string
)

func addController(args []string) {
	// check for args by count. need two arguments.
	if len(args) == 0 {
		errorf("The app path and controller name were not supplied. Run 'revel help add' for usage.\n")
	}
	if len(args) != 2 {
		errorf("The incorrect number of arguments was supplied. Run 'revel help add' for usage.\n")
	}

	controller = args[1]

	// checking and setting go paths
	initGoPaths()

	// create the path to the application
	createApplicationPath(args[0])

	// check if the application path exists
	checkApplicationPath()

	// check if the path to the controllers directory
	// exists
	checkControllersDirectory()

	// create the new controller file
	createControllerFile()

	// create the view directory
	createControllerViews()

	fmt.Println("Done!")
}

func createApplicationPath(appName string) {
	appPath = filepath.Join(srcRoot, filepath.FromSlash(appName))
}

func checkApplicationPath() {
	fmt.Printf("Adding controller: %s.go\n", controller)
	fmt.Printf("Application Path: %s\n", appPath)

	_, err := os.Stat(appPath)
	if os.IsNotExist(err) {
		errorf("Abort: Application path %s does not exist.\n", appPath)
	}

	revelPkg, err = build.Import(revel.REVEL_IMPORT_PATH, "", build.FindOnly)
	if err != nil {
		errorf("Abort: Could not find Revel source code: %s\n", err)
	}

	revelPath = revelPkg.ImportPath

	// now check if we can access the app directory
	// under the app path
	_, err = os.Stat(appPath + "/app")
	if os.IsNotExist(err) {
		errorf("Abort: Could not access the /app directory: %s\n", appPath+"/app")
	}
}

func checkControllersDirectory() {
	// see if we can access the controllers directory
	// if we can, do nothing. if we cannot, create it.
	controllersDirectory := appPath + "/app/controllers"
	_, err := os.Stat(controllersDirectory)
	if os.IsNotExist(err) {
		fmt.Printf("Creating %s\n", controllersDirectory)
		err2 := os.Mkdir(controllersDirectory, 0755)
		if err2 != nil {
			fmt.Printf("Abort: Could not create the controllers directory: %s\n", controllersDirectory)
		}
	}
}

func getPathToTemplate() string {
	_, dir, _, _ := runtime.Caller(1)
	dir, _ = filepath.Split(dir)

	templateFile := filepath.Join(dir, "new_controller.template")

	return templateFile
}

// create the new controller file
func createControllerFile() {
	// get the path to the controller template
	templateFile := getPathToTemplate()
	tmpl, err := template.ParseFiles(templateFile)
	panicOnError(err, "Could not parse template file: new_controller.template.")

	controllerFile := "/app/controllers/" + controller + ".go"
	controllerPath := filepath.Join(appPath, controllerFile)

	_, err = os.Stat(controllerPath)
	if err == nil {
		errorf("Abort: The controller %s already exists.", controllerPath)
	}

	f, err := os.Create(controllerPath)
	panicOnError(err, "Could not create "+controllerPath)

	tmplData := map[string]interface{}{
		"Controller": strings.Title(controller),
		"RevelPath":  revelPath,
	}

	err = tmpl.Execute(f, tmplData)

	err = f.Close()
	panicOnError(err, "Failed to close "+f.Name())
}

// create the views directory for the controller
func createControllerViews() {
	viewsPath := "/app/views/" + strings.Title(controller)
	viewsPath = filepath.Join(appPath, viewsPath)

	// check if the views directory exists
	_, err := os.Stat(viewsPath)
	if err == nil {
		errorf("Abort: The views directory %s already exists.", viewsPath)
	}

	err = os.Mkdir(viewsPath, 0755)
	if err != nil {
		panicOnError(err, "Could not create the views directory: "+viewsPath)
	}
}
