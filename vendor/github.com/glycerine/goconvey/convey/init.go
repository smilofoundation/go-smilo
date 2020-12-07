package convey

import (
	"flag"
	//"fmt"
	"os"

	"github.com/glycerine/goconvey/convey/reporting"
	"github.com/jtolds/gls"
)

type GoConveyConfig struct {
	Myflags *flag.FlagSet

	json   bool
	silent bool
	story  bool
	chatty bool
	match  string

	verboseEnabled bool // = flagFound("-test.v=true")
	storyDisabled  bool // = flagFound("-story=false")

	testReporter reporting.Reporter
}

var Cfg = &GoConveyConfig{}

func init() {

	declareFlags(Cfg)

	ctxMgr = gls.NewContextManager()

	//fmt.Printf("done with init.go init()\n")
}

func declareFlags(cfg *GoConveyConfig) {

	f := flag.NewFlagSet("goConvey", flag.ContinueOnError)
	cfg.Myflags = f

	f.BoolVar(&cfg.json, "json", false, "When true, emits results in JSON blocks. Default: 'false'")
	f.BoolVar(&cfg.silent, "silent", false, "When true, all output from GoConvey is suppressed.")
	f.BoolVar(&cfg.story, "story", false, "When true, emits story output, otherwise emits dot output. When not provided, this flag mirrors the value of the '-test.v' flag")

	f.BoolVar(&cfg.chatty, "test.v", false, "verbose: print additional output")
	f.StringVar(&cfg.match, "test.run", "", "run only tests and examples matching `regexp`")

	f.Parse(os.Args[1:])

	cfg.verboseEnabled = flagFound("-test.v=true")
	cfg.storyDisabled = flagFound("-story=false")

	if !cfg.story && !cfg.storyDisabled {
		cfg.story = cfg.verboseEnabled
	}

	// FYI: flag.Parse() is called from the testing package.
}

func buildReporter() reporting.Reporter {
	switch {
	case Cfg.testReporter != nil:
		return Cfg.testReporter
	case Cfg.json:
		return reporting.BuildJsonReporter()
	case Cfg.silent:
		return reporting.BuildSilentReporter()
	case Cfg.story:
		return reporting.BuildStoryReporter()
	default:
		return reporting.BuildDotReporter()
	}
}

var (
	ctxMgr *gls.ContextManager

	// only set by internal tests

)

// flagFound parses the command line args manually for flags defined in other
// packages. Like the '-v' flag from the "testing" package, for instance.
func flagFound(flagValue string) bool {
	for _, arg := range os.Args {
		if arg == flagValue {
			return true
		}
	}
	return false
}
