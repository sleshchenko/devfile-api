package tests

import (
	"bytes"
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/santhosh-tekuri/jsonschema"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Structure of json files providing schema to test and tests to run
type TestJson struct {
	SchemaFile    string   `json:"SchemaFile"`
	SchemaVersion string   `json:"SchemaVersion"`
	Tests         []string `json:"Tests"`
}

// Structure of a set of tests to run
type TestsToRun struct {
	Tests []TestToRun `json:"Tests"`
}

// Structure for an individual test
type TestToRun struct {
	FileName      string   `json:"FileName"`
	ExpectOutcome string   `json:"ExpectOutcome"`
	Disabled      bool     `json:"Disabled"`
	Files         []string `json:"Files"`
}

const testDir = "../../../"
const schemaDir = "../../../../"
const jsonDir = "./json/v200/"

const tempRootDir = "./tmp/"
const tempDir = tempRootDir + "v200/"

func Test_API_200(t *testing.T) {
	err := prepareTmpDir()
	if err != nil {
		t.Fatalf("Can't prepare tmpDir '%s`.Cause: %v", jsonDir, err)
	}

	// Read the content of the json directory to find test files
	files, err := ioutil.ReadDir(jsonDir)
	if err != nil {
		t.Fatalf("Error finding test json files in : %s :  %v", jsonDir, err)
	}
	var testFiles []os.FileInfo
	for _, f := range files {
		// if the file begins with test- and ends .json it can be processed
		if strings.HasPrefix(f.Name(), "test-") && strings.HasSuffix(f.Name(), ".json") {
			testFiles = append(testFiles, f)
		}
	}
	combinedTests := 0
	combinedPasses := 0
	combinedSkipped := 0
	for _, testJsonFile := range testFiles {
		testJsonContent, err := readTestJson(testJsonFile)
		if err != nil {
			t.Errorf("  FAIL : Failed to read and parse %s. Cause: %s", testJsonFile.Name(), err)
			continue
		}

		t.Logf("INFO : File %s : SchemaFile : %s , SchemaVersion : %s", testJsonFile.Name(), testJsonContent.SchemaFile, testJsonContent.SchemaVersion)

		// Prepare the schema file
		compiler := jsonschema.NewCompiler()
		compiler.Draft = jsonschema.Draft7
		schema, err := compiler.Compile(filepath.Join(schemaDir, testJsonContent.SchemaFile))
		if err != nil {
			t.Fatalf("  FAIL : Schema compile failed : %s: %v", testJsonContent.SchemaFile, err)
		}

		// create the temp directory to hold the generated yaml files
		testTempDir := tempDir
		testTempDir += strings.Split(testJsonFile.Name(), ".")[0]
		os.Mkdir(testTempDir, 0755)

		totalTests := 0
		passTests := 0
		skippedTests := 0

		// for each of the test files specified
		for _, testJsonFile := range testJsonContent.Tests {
			testsToRunContent, err := readTestsToRun(testJsonFile)
			if err != nil {
				t.Errorf("Failed to parse test file '%s'. Cause: %v", testJsonFile, err)
				continue
			}

			// For each test defined in the test file
			for _, testToRun := range testsToRunContent.Tests {
				totalTests++
				if testToRun.Disabled {
					skippedTests++
					continue
				}

				// Open the file to contains the generated test yaml
				f, err := os.OpenFile(filepath.Join(testTempDir, testToRun.FileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					t.Errorf("FAIL : Failed to open %s : %v", filepath.Join(testTempDir, testToRun.FileName), err)
					continue
				}

				// If test requires a schema write it to the yaml file
				if testJsonContent.SchemaVersion != "" {
					f.WriteString("schemaVersion: " + testJsonContent.SchemaVersion + "\n")
				}

				testYamlComplete := true
				// Now add each of the yaml snippets used the make the yaml file for test
				for j := 0; j < len(testToRun.Files); j++ {
					// Read the snippet
					data, err := ioutil.ReadFile(filepath.Join(testDir, testToRun.Files[j]))
					if err != nil {
						t.Errorf("FAIL: failed reading %s: %v", filepath.Join(testDir, testToRun.Files[j]), err)
						testYamlComplete = false
						continue
					}
					if j > 0 {
						// Ensure appropriate line breaks
						f.WriteString("\n")
					}

					// Add snippet to yaml file
					f.Write(data)
				}

				if !testYamlComplete {
					f.Close()
					continue
				}

				// Read the created yaml file, ready for converison to json
				data, err := ioutil.ReadFile(filepath.Join(testTempDir, testToRun.FileName))
				if err != nil {
					t.Errorf("  FAIL: unable to read %s: %v", testToRun.FileName, err)
					f.Close()
					continue
				}

				f.Close()

				// Convert the yaml file to json
				yamldoc, err := yaml.YAMLToJSON(data)
				if err != nil {
					t.Errorf("  FAIL : %s : failed to convert to json : %v", testToRun.FileName, err)
					continue
				}

				// validate the test yaml against the schema
				if err = schema.Validate(bytes.NewReader(yamldoc)); err != nil {
					if testToRun.ExpectOutcome == "PASS" {
						t.Errorf("  FAIL : %s : %s : Validate failure : %s", testToRun.FileName, testJsonContent.SchemaFile, err)
					} else if testToRun.ExpectOutcome == "" {
						t.Errorf("  FAIL : %s : No expected ouctome was set : %s  got : %s", testToRun.FileName, testToRun.ExpectOutcome, err.Error())
					} else if !strings.Contains(err.Error(), testToRun.ExpectOutcome) {
						t.Errorf("  FAIL : %s : %s : Did not fail as expected : %s  got : %s", testToRun.FileName, testJsonContent.SchemaFile, testToRun.ExpectOutcome, err.Error())
					} else {
						passTests++
						t.Logf("PASS : %s : %s: %s", testToRun.FileName, testJsonContent.SchemaFile, testToRun.ExpectOutcome)
					}
				} else if testToRun.ExpectOutcome == "" {
					t.Errorf("  FAIL : %s : devfile was valid - No expected ouctome was set.", testToRun.FileName)
				} else if testToRun.ExpectOutcome != "PASS" {
					t.Errorf("  FAIL : %s : devfile was valid - Expected Error not found :  %s", testToRun.FileName, testToRun.ExpectOutcome)
				} else {
					passTests++
					t.Logf("PASS : %s : %s", testToRun.FileName, testJsonContent.SchemaFile)
				}
				f.Close()
			}
		}
		t.Logf("PASS %s : \n"+
			" %d tests;\n"+
			" %d tests passed\n"+
			" %d tests skipped.", testJsonFile.Name(), combinedPasses, combinedTests, skippedTests)
		combinedTests += totalTests
		combinedPasses += passTests
		combinedSkipped += skippedTests

	}

	if combinedTests != combinedPasses+combinedSkipped {
		t.Errorf("\nOVERALL FAIL : %d of %d tests failed.", combinedTests-combinedPasses-combinedSkipped, combinedTests)
	} else {
		t.Logf("\n--------------------\n"+
			"OVERALL PASS: \n"+
			" %d tests\n"+
			" %d tests passed\n"+
			" %d tests skipped.", combinedTests, combinedPasses, combinedSkipped)
	}
}

func readTestsToRun(testPath string) (*TestsToRun, error) {
	// Open the json file which defines the tests to run
	testsToRunJson, err := os.Open(filepath.Join(jsonDir, testPath))
	if err != nil {
		return nil, err
	}

	// Read contents of the json file which defines the tests to run
	byteValue, err := ioutil.ReadAll(testsToRunJson)
	if err != nil {
		return nil, err
	}
	testsToRunJson.Close()

	testsToRunContent := &TestsToRun{}

	// Unmarshall the contents of the json file which defines the tests to run for each test
	err = json.Unmarshal(byteValue, testsToRunContent)
	if err != nil {
		return nil, err
	}

	return testsToRunContent, nil
}

func readTestJson(testJsonFile os.FileInfo) (*TestJson, error) {
	// Open the json file which defines the tests to run
	testJson, err := os.Open(filepath.Join(jsonDir, testJsonFile.Name()))
	if err != nil {
		return nil, err
	}

	// Read contents of the json file which defines the tests to run
	byteValue, err := ioutil.ReadAll(testJson)
	if err != nil {
		return nil, err
	}

	testJsonContent := &TestJson{}
	// Unmarshall the contents of the json file which defines the tests to run for each test
	err = json.Unmarshal(byteValue, testJsonContent)
	if err != nil {
		return nil, err
	}

	testJson.Close()

	return testJsonContent, nil
}

func prepareTmpDir() (err error) {
	// Clear the temp directory if it exists
	if _, err = os.Stat(tempRootDir); err == nil {
		err := os.RemoveAll(tempRootDir)
		if err != nil {
			return err
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = os.Mkdir(tempRootDir, 0755)
	if err != nil {
		return err
	}

	err = os.Mkdir(tempDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

// Users struct which contains
// an array of users
type Command struct {
	ID            string `yaml:"ID"`
	SchemaVersion string `yam:"SchemaVersion"`
}

func Test_WriteYaml(t *testing.T) {
	command := Command{}

	command.ID = "TestYaml"
	command.SchemaVersion = "2.0.0"

	c, err := yaml.Marshal(&command)

	f, err := os.Create("/tmp/data")

	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = ioutil.WriteFile("command.yaml", c, 0644)
	if err != nil {
		t.Fatal(err)
	}
}
