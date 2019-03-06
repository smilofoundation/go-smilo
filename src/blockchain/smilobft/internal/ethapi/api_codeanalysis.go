package ethapi

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type codeAnalysisAPI struct {
	// This lock guards the codeanalysis path set through the API.
	// It also ensures that only one codeanalysis process is used at
	// any time.
	mu               sync.Mutex
	codeanalysisPath string
}

type PublicCodeAnalysisAPI codeAnalysisAPI
type CodeAnalysisAdminAPI codeAnalysisAPI

type AnalysysResult struct {
	Vulnerable bool
	Output     string
}

type SmiloAnalysisArgs struct {
	Code       string
	Arguments  string
	Abi        string
	Hash       string
	List       bool
	Disassm    bool
	SingleStep bool
	Cfg        bool
	CfgFull    bool
	Decompile  bool
	Debug      bool
	Silent     bool
}

func parseArgs(key, value string) string {
	return fmt.Sprintf("--%s=%s", key, value)
}

func (api PublicCodeAnalysisAPI) CodeAnalysis(args SmiloAnalysisArgs) (*AnalysysResult, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	var (
		stderr, stdout bytes.Buffer
		arguments      []string
	)
	arguments = append(arguments, parseArgs("code", args.Code))

	if args.Arguments != "" {
		arguments = append(arguments, parseArgs("arguments", args.Arguments))
	}
	if args.Abi != "" {
		arguments = append(arguments, parseArgs("abi", args.Abi))
	}
	if args.Hash != "" {
		arguments = append(arguments, parseArgs("hash", args.Hash))
	}
	if args.List {
		arguments = append(arguments, parseArgs("list", ""))
	}
	if args.Disassm {
		arguments = append(arguments, parseArgs("disassemble", ""))
	}
	if args.SingleStep {
		arguments = append(arguments, parseArgs("single-step", ""))
	}
	if args.Cfg {
		arguments = append(arguments, parseArgs("cfg", ""))
	}
	if args.CfgFull {
		arguments = append(arguments, parseArgs("cfg-full", ""))
	}
	if args.Decompile {
		arguments = append(arguments, parseArgs("decompile", ""))
	}
	if args.Debug {
		arguments = append(arguments, parseArgs("debug", ""))
	}
	cmd := exec.Command(api.codeanalysisPath, arguments...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	out := stdout.String()
	pr := new(AnalysysResult)
	pr.Vulnerable = strings.Contains(out, "[0;31m")
	if !args.Silent {
		pr.Output = out
	}
	return pr, nil
}

// SetSolc sets the Solidity compiler path to be used by the node.
func (api *CodeAnalysisAdminAPI) SetSmiloAnalysis(path string) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	err := smiloAnalysisCheck(path)
	if err != nil {
		return err
	}
	api.codeanalysisPath = path
	return nil
}

// SolidityVersion runs codeanalysisPath and parses its version output.
func smiloAnalysisCheck(codeanalysisPath string) error {
	if codeanalysisPath == "" {
		codeanalysisPath = "smilo-code-analysis"
	}
	var out bytes.Buffer
	cmd := exec.Command(codeanalysisPath, "--version")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
