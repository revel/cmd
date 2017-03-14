package builder

import (
    "fmt"
    "os"
    "path/filepath"
    "os/exec"
    "bytes"
    "github.com/revel/cmd/revel/util"
    "go/build"
)
type SkeletonApp struct {
    BaseNewApp
}

func NewSkeleton(args []string) {
    app := &SkeletonApp{}
    app.newApp(args)
    // checking and setting skeleton
    app.setSkeletonPath(args)

    // copy files to new app directory
    app.copyNewAppFiles()

    // goodbye world
    fmt.Fprintln(os.Stdout, "Your application is ready:\n  ", app.Config.appPath)
    fmt.Fprintln(os.Stdout, "\nYou can run it with:\n   revel run", app.Config.importPath)

}
func(i *SkeletonApp)  setSkeletonPath(args []string) {
    var err error
    if len(args) == 2 { // user specified
        skeletonName := args[1]
        _, err = build.Import(skeletonName, "", build.FindOnly)
        if err != nil {
            // Execute "go get <pkg>"
            getCmd := exec.Command(i.Config.gocmd, "get", "-d", skeletonName)
            fmt.Println("Exec:", getCmd.Args)
            getOutput, err := getCmd.CombinedOutput()

            // check getOutput for no buildible string
            bpos := bytes.Index(getOutput, []byte("no buildable Go source files in"))
            if err != nil && bpos == -1 {
                i.errorf("Abort: Could not find or 'go get' Skeleton  source code: %s\n%s\n", getOutput, skeletonName)
            }
        }
        // use the
        i.Config.skeletonPath = filepath.Join(i.Config.srcRoot, skeletonName)

    } else {
        // use the revel default
        revelCmdPkg, err = build.Import(util.RevelCmdImportPath, "", build.FindOnly)
        if err != nil {
            i.errorf("Abort: Could not find Revel Cmd source code: %s\n", err)
        }

        i.Config.skeletonPath = filepath.Join(revelCmdPkg.Dir, "revel", "skeleton")
    }
}

func(i *SkeletonApp)  copyNewAppFiles() {
    var err error
    if err = os.MkdirAll(i.Config.appPath, 0777);err!=nil {
        i.errorf("Failed to create directory " + i.Config.appPath)
    }

    _ = util.MustCopyDir(i.Config.appPath, i.Config.skeletonPath, map[string]interface{}{
        // app.conf
        "AppName":  i.Config.appName,
        "BasePath": i.Config.basePath,
        "Secret":   i.generateSecret(),
    })

    // Dotfiles are skipped by mustCopyDir, so we have to explicitly copy the .gitignore.
    gitignore := ".gitignore"
    util.MustCopyFile(filepath.Join(i.Config.appPath, gitignore), filepath.Join(i.Config.skeletonPath, gitignore))

}
