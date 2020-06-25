package app

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/SimonBaeumer/commander/pkg/output"
	"github.com/SimonBaeumer/commander/pkg/runtime"
	"github.com/SimonBaeumer/commander/pkg/suite"
)

var out output.OutputWriter

// TestCommand executes the test argument
// testPath is the path to the test suite config, it can be a dir or file
// titleFilterTitle is the title of test which should be executed, if empty it will execute all tests
// ctx holds the command flags. If directory scanning is enabled with --dir it is
// not supported to filter tests, therefore testFilterTitle is an empty string
func TestCommand(testPath string, testFilterTitle string, ctx AddCommandContext) error {
	if ctx.Verbose == true {
		log.SetOutput(os.Stdout)
	}

	out = output.NewCliOutput(!ctx.NoColor)

	if testPath == "" {
		testPath = CommanderFile
	}

	var result output.Result
	var err error
	if ctx.Dir {
		if testFilterTitle != "" {
			return fmt.Errorf("Test may not be filtered when --dir is enabled")
		}
		fmt.Println("Starting test against directory: " + testPath + "...")
		fmt.Println("")
		result, err = testDir(testPath)
	} else {
		fmt.Println("Starting test file " + testPath + "...")
		fmt.Println("")
		result, err = testFile(testPath, "", testFilterTitle)
	}

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	if !out.PrintSummary(result) {
		return fmt.Errorf("Test suite failed, use --verbose for more detailed output")
	}

	return nil
}

func testDir(directory string) (output.Result, error) {
	result := output.Result{}

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return result, fmt.Errorf(err.Error())
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		path := path.Join(directory, f.Name())
		newResult, err := testFile(path, f.Name(), "")
		if err != nil {
			return result, err
		}

		result = convergeResults(result, newResult)
	}

	return result, nil
}

func convergeResults(result output.Result, new output.Result) output.Result {
	result.TestResults = append(result.TestResults, new.TestResults...)
	result.Failed += new.Failed
	result.Duration += new.Duration

	return result
}

func testFile(filePath string, fileName string, title string) (output.Result, error) {
	content, err := readFile(filePath)
	if err != nil {
		return output.Result{}, fmt.Errorf("Error " + err.Error())
	}

	s := suite.ParseYAML(content, fileName)
	result, err := execute(s, title)
	if err != nil {
		return output.Result{}, fmt.Errorf("Error " + err.Error())
	}

	return result, nil
}

func execute(s suite.Suite, title string) (output.Result, error) {
	tests := s.GetTests()
	// Filter tests if test title was given
	if title != "" {
		test, err := s.GetTestByTitle(title)
		if err != nil {
			return output.Result{}, err
		}
		tests = []runtime.TestCase{test}
	}

	r := runtime.NewRuntime(&out, s.Nodes...)
	result := r.Start(tests)

	return result, nil
}

func readFile(filePath string) ([]byte, error) {
	f, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: no such file or directory", filePath)
	}

	if f.IsDir() {
		return nil, fmt.Errorf("%s: is a directory\nUse --dir to test directories with multiple test files", filePath)
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return content, nil
}
