package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	. "github.com/ForceCLI/force/error"
	. "github.com/ForceCLI/force/lib"
	"github.com/spf13/cobra"
)

func init() {
	// Deploy options
	importCmd.Flags().BoolP("rollbackonerror", "r", false, "roll back deployment on error")
	importCmd.Flags().BoolP("runalltests", "t", false, "run all tests (equivalent to --testlevel RunAllTestsInOrg)")
	importCmd.Flags().StringP("testlevel", "l", "NoTestRun", "test level")
	importCmd.Flags().BoolP("checkonly", "c", false, "check only deploy")
	importCmd.Flags().BoolP("purgeondelete", "p", false, "purge metadata from org on delete")
	importCmd.Flags().BoolP("allowmissingfiles", "m", false, "set allow missing files")
	importCmd.Flags().BoolP("autoupdatepackage", "u", false, "set auto update package")
	importCmd.Flags().BoolP("ignorewarnings", "i", false, "ignore warnings")
	importCmd.Flags().StringSliceP("test", "", []string{}, "Test(s) to run")

	// Output options
	importCmd.Flags().BoolP("ignorecoverage", "w", false, "suppress code coverage warnings")
	importCmd.Flags().BoolP("suppressunexpected", "U", true, `suppress "An unexpected error occurred" messages`)
	importCmd.Flags().BoolP("quiet", "q", false, "only output failures")
	importCmd.Flags().CountP("verbose", "v", "give more verbose output")
	importCmd.Flags().StringP("reporttype", "f", "text", "report type format (text or junit)")

	importCmd.Flags().StringP("directory", "d", "src", "relative path to package.xml")

	importCmd.Flags().BoolVarP(&errorOnTestFailure, "erroronfailure", "E", true, "exit with an error code if any tests fail")

	RootCmd.AddCommand(importCmd)
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import metadata from a local directory",
	Example: `
  force import
  force import -directory=my_metadata -c -r -v
  force import -checkonly -runalltests
`,
	Run: func(cmd *cobra.Command, args []string) {
		options := getDeploymentOptions(cmd)
		srcDir := sourceDir(cmd)

		displayOptions := getDeploymentOutputOptions(cmd)

		runImport(srcDir, options, displayOptions)
	},
	Args: cobra.MaximumNArgs(0),
}

var (
	errorOnTestFailure bool
)

func sourceDir(cmd *cobra.Command) string {
	directory, _ := cmd.Flags().GetString("directory")

	wd, _ := os.Getwd()
	var dir string
	usr, err := user.Current()

	//Manually handle shell expansion short cut
	if err != nil {
		if strings.HasPrefix(directory, "~") {
			ErrorAndExit("Cannot determine tilde expansion, please use relative or absolute path to directory.")
		} else {
			dir = directory
		}
	} else {
		if strings.HasPrefix(directory, "~") {
			dir = strings.Replace(directory, "~", usr.HomeDir, 1)
		} else {
			dir = directory
		}
	}

	root := filepath.Join(wd, dir)

	// Check for absolute path
	if filepath.IsAbs(dir) {
		root = dir
	}
	return root
}

func runImport(root string, options ForceDeployOptions, displayOptions *deployOutputOptions) {
	files := make(ForceMetadataFiles)
	if _, err := os.Stat(filepath.Join(root, "package.xml")); os.IsNotExist(err) {
		ErrorAndExit(" \n" + filepath.Join(root, "package.xml") + "\ndoes not exist")
	}

	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			if f.Name() != ".DS_Store" {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					ErrorAndExit(err.Error())
				}
				files[strings.Replace(path, fmt.Sprintf("%s%s", root, string(os.PathSeparator)), "", -1)] = data
			}
		}
		return nil
	})
	if err != nil {
		ErrorAndExit(err.Error())
	}

	err = deploy(force, files, &options, displayOptions)
	if err == nil && displayOptions.reportFormat == "text" && !displayOptions.quiet {
		fmt.Printf("Imported from %s\n", root)
	}
	if err != nil {
		ErrorAndExit(err.Error())
	}
}
