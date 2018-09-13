package model

type (
	// The Revel command type
	COMMAND int

	// The Command config for the line input
	CommandConfig struct {
		Index        COMMAND  // The index
		Verbose      bool     `short:"v" long:"debug" description:"If set the logger is set to verbose"` // True if debug is active
		HistoricMode bool     `long:"historic-run-mode" description:"If set the runmode is passed a string not json"` // True if debug is active
		ImportPath   string   // The import path (converted from various commands)
		GoPath       string   // The GoPath
		GoCmd        string   // The full path to the go executable
		SrcRoot      string   // The source root
		AppPath      string   // The application path
		AppName      string   // The applicaiton name
		BasePath     string   // The base path
		SkeletonPath string   // The skeleton path
		BuildFlags   []string `short:"X" long:"build-flags" description:"These flags will be used when building the application. May be specified multiple times, only applicable for Build, Run, Package, Test commands"`
		// The new command
		New struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
			Skeleton   string `short:"s" long:"skeleton" description:"Path to skeleton folder (Must exist on GO PATH)" required:"false"`
			Vendored   bool   `short:"V" long:"vendor" description:"True if project should contain a vendor folder to be initialized. Creates the vendor folder and the 'Gopkg.toml' file in the root"`
			Run        bool   `short:"r" long:"run" description:"True if you want to run the application right away"`
		} `command:"new"`
		// The build command
		Build struct {
			TargetPath string `short:"t" long:"target-path" description:"Path to target folder. Folder will be completely deleted if it exists" required:"true"`
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
		} `command:"build"`
		// The run command
		Run struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			Port       string `short:"p" long:"port" description:"The port to listen"`
			NoProxy    bool   `short:"n" long:"no-proxy" description:"True if proxy server should not be started. This will only update the main and routes files on change"`
		} `command:"run"`
		// The package command
		Package struct {
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
		} `command:"package"`
		// The clean command
		Clean struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
		} `command:"clean"`
		// The test command
		Test struct {
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
			Function   string `short:"f" long:"suite-function" description:"The suite.function"`
		} `command:"test"`
		// The version command
		Version struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"false"`
		} `command:"version"`
	}
)
