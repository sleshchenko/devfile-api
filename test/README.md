# API Tests

The API tests are intended to provide a comprehensive verification of the devfile schemas. This includes:
- Ensuring every possible attribute is valid.
- Ensuring all optional attributes are indeed optional.
- Ensuring any possible specification errors are invalidated by the schema. For example:
    - Missing mandatory attributes.
    - Mutiple use of a one-of attribute.
    - atttribute values of the wrong type.


## Test structure

- `test/v200/devfiles` : contains yaml snippets which are used to generate yaml files for the tests. The names of the sub-directories and files should reflect their purpose.
- `test/go/src/tests/api-v200-test.go` : the go unit test program.
- `test/go/src/tests/json/v200` :  contains the json files which define the tests which the test program will run:
    - `test-xxxxxxx.json` : these files are the top level json files, they define the schema to verify and the test files to run.
    - `xxxxxx-tests.json` : these are the test files which contain individual tests which provide the yaml snippets to combine and the expected result.

## Running tests

from the test/go/src/test directory run 
- `go test -v`

The test will read each of the test-xxxxxx.json files and run the tests defined within. The generated .yaml files used for the tests are created in a `tmp/v200/test-xxxxxx/` directory. These files are not deleted when the test finishes so they can be used to assess any errors, however they will be deleted by a subsequent run of the test. Running the test with the -v option ensures you see a full list of passes and failures. 

## Adding Tests

### add a test for a new schema file

1. create a new `test/go/src/tests/json/v200/test-<schema name>.json` file for the schema. In the json file  specify the location of the schema to test (relative to the root directory of the repository), and the list of the existing tests to use. If the generated yaml files require a schemaVersion attribute include its value in the json file. see - *link to sample schema to be added*
1. run the test

### add a test for a schema changes

1. Modify an an existing yaml snippet or create a new one.
2. If approprate create a new snippet for any possible error cases, for example to omit a required attribute.
3. If a new yaml snippet was created add a test which uses the snippet to the appropriate `json/xxxxxx-tests.json` file. Be careful to ensure the FileName set for the test is unique for all tests - this is the name used for the yaml file which is generated for the test. For failure scenarios you may need to run the test first to set the outcome correctly. 
4. If a new  `json/xxxxxx-tests.json` file is created, any existing `test-xxxxxxx.json` files must be updated to use the new file.


