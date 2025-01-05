package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	externalTaskIconName     = "externalTaskIcon"
	internalTaskIconName     = "internalTaskIcon"
	unknownTaskIconName      = "unknownTaskIcon"
	varIconName              = "varIcon"
	includedTaskfileIconName = "includedTaskfileIcon"
)

var includesToIncludedTasks = make(map[string]map[string]struct{})

func main() {
	rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "taskfile2d2 [Taskfile.yml] [Taskfile.yml.d2]",
	Short: "taskfile2d2 is a tool that generates a Terrastruct D2 diagram file from a Taskfile",
	Example: `# Examples for passing input as argument:
# Passing the input as argument without specifying the output, will write the output to "ARG1.d2" where ARG1 is the first argument (Taskfile.yml)
taskfile2d2 Taskfile.yml

# Passing the input as argument, while also specifying the output file
taskfile2d2 Taskfile.yml out.d2


# Examples for passing input via standard input. The output to the standard output:
# With the "cat" command
cat Taskfile.yml | taskfile2d2 > output.d2

# With stdin redirection, avoiding the "cat" command
taskfile2d2 < Taskfile.yml > output.d2

# Pipe the Taskfile from a remote source
curl -s http://example.com/Taskfile.yml | taskfile2d2 > output.d2

# Print the D2 content to terminal
taskfile2d2 > output.d2
`,
	Version: "0.0.1",
	Args:    cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if stdin is connected to a terminal
		fileInfo, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("error checking stdin: %w", err)
		}

		if (fileInfo.Mode() & os.ModeNamedPipe) == 0 {
			if len(args) == 0 {
				return cmd.Help()
			}
			taskFile, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			d2, err := TaskfileToD2(taskFile)
			if err != nil {
				return err
			}
			var d2OutFilePath string
			if len(args) >= 2 {
				d2OutFilePath = args[1]
			} else {
				d2OutFilePath = args[0] + ".d2"
			}
			os.WriteFile(d2OutFilePath, []byte(d2), fs.ModePerm)
		} else {
			err = ProcessIO(func(b []byte) ([]byte, error) {
				d2, err := TaskfileToD2(b)
				if err != nil {
					return nil, err
				}
				return []byte(d2), nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	},
}

// func initConfig() {
// 	if cfgFile != "" {
// 		// Use config file from the flag.
// 		viper.SetConfigFile(cfgFile)
// 	} else {
// 		// Find home directory.
// 		home, err := os.UserHomeDir()
// 		cobra.CheckErr(err)

// 		// Search config in home directory with name ".cobra" (without extension).
// 		viper.AddConfigPath(home)
// 		viper.SetConfigType("yaml")
// 		viper.SetConfigName(".cobra")
// 	}

// 	viper.AutomaticEnv()

// 	if err := viper.ReadInConfig(); err == nil {
// 		fmt.Println("Using config file:", viper.ConfigFileUsed())
// 	}
// }

type Task struct {
	Desc     string
	Summary  string
	Silent   bool
	Internal bool
	Requires struct {
		Vars []any
	}
	Vars map[string]any
	Deps []any
	Cmd  any
	Cmds []any
}
type Taskfile struct {
	Includes map[string]any
	Version  string
	Vars     map[string]any
	Tasks    map[string]Task
}

func (tf *Taskfile) GetIncludes() (result []string) {
	for key := range tf.Includes {
		result = append(result, key)
	}
	return
}

func (t *Task) GetDepCalls() (result []TaskCall) {
	for _, dep := range t.Deps {
		var taskCall TaskCall
		switch dep := dep.(type) {
		case string:
			taskCall.TaskName = dep
		case map[string]any:
			taskCall.TaskName = dep["task"].(string)
			passedVars, isVarMap := dep["vars"].(map[string]any)
			if isVarMap {
				for _, passedVarName := range slices.Sorted(maps.Keys(passedVars)) {
					taskCall.Vars = append(taskCall.Vars, Variable{
						Name:  passedVarName,
						Value: passedVars[passedVarName],
					})
				}
			}
		default:
			panic("")
		}
		result = append(result, taskCall)
	}
	return
}
func (t *Task) GetCmds() []any {
	if t.Cmd != nil && t.Cmds != nil {
		log.Fatal("task cannot have both cmd and cmds")
	}
	if t.Cmd == nil {
		return t.Cmds
	} else {
		return []any{t.Cmd}
	}

}

type Variable struct {
	Name  string
	Value any
}
type TaskCall struct {
	TaskName string
	Vars     []Variable
}

func (t *Task) GetCalls() (result []TaskCall) {
	for _, cmd := range t.GetCmds() {
		if typedCmd, isMap := cmd.(map[string]any); isMap {
			taskName, hasTaskCall := typedCmd["task"].(string)
			if hasTaskCall {
				taskCall := TaskCall{
					TaskName: taskName,
				}
				passedVars, isVarMap := typedCmd["vars"].(map[string]any)
				if isVarMap {
					for _, passedVarName := range slices.Sorted(maps.Keys(passedVars)) {
						taskCall.Vars = append(taskCall.Vars, Variable{
							Name:  passedVarName,
							Value: passedVars[passedVarName],
						})
					}
				}
				result = append(result, taskCall)
			}
		}
	}
	return
}

type RequiredVariable struct {
	Name string
	Enum []string
}

func (t *Task) GetRequiredVars() (result []RequiredVariable) {
	for _, variable := range t.Requires.Vars {
		switch variable := variable.(type) {
		case string:
			result = append(result, RequiredVariable{Name: variable})
		case map[string]any:
			requiredVariable := RequiredVariable{
				Name: variable["name"].(string),
			}
			for _, enum := range variable["enum"].([]any) {
				requiredVariable.Enum = append(requiredVariable.Enum, enum.(string))
			}
			result = append(result, requiredVariable)
		default:
			panic("")
		}
	}
	return
}
func TaskfileToD2(taskfileYaml []byte) (string, error) {
	var taskfile Taskfile
	err := yaml.Unmarshal(taskfileYaml, &taskfile)
	if err != nil {
		return "", err
	}
	if taskfile.Version != "3" {
		log.Fatal("Only version 3 Taskfiles are supported")
	}
	d2Writer := NewD2Writer()
	d2Vars := fmt.Sprintf(`{
  %s: %s
  %s: %s
  %s: %s
  %s: %s
  %s: %s
}`,
		externalTaskIconName,
		`data:image/svg+xml,%3C%3Fxml version='1.0' encoding='utf-8'%3F%3E%3Csvg fill='%23000000' width='800px' height='800px' viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M5 22h14c1.103 0 2-.897 2-2V5c0-1.103-.897-2-2-2h-2a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1H5c-1.103 0-2 .897-2 2v15c0 1.103.897 2 2 2zM5 5h2v2h10V5h2v15H5V5z'/%3E%3Cpath d='m11 13.586-1.793-1.793-1.414 1.414L11 16.414l5.207-5.207-1.414-1.414z'/%3E%3C/svg%3E`,
		internalTaskIconName,
		`data:image/svg+xml,%3C%3Fxml version='1.0' encoding='utf-8'%3F%3E%3Csvg fill='%23000000' width='800px' height='800px' viewBox='0 0 24 24' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M3 20c0 1.103.897 2 2 2h14c1.103 0 2-.897 2-2V5c0-1.103-.897-2-2-2h-2a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1H5c-1.103 0-2 .897-2 2v15zM5 5h2v2h10V5h2v15H5V5z'/%3E%3Cpath d='M14.292 10.295 12 12.587l-2.292-2.292-1.414 1.414 2.292 2.292-2.292 2.292 1.414 1.414L12 15.415l2.292 2.292 1.414-1.414-2.292-2.292 2.292-2.292z'/%3E%3C/svg%3E`,
		unknownTaskIconName,
		`data:image/svg+xml,%3C%3Fxml version='1.0' encoding='utf-8'%3F%3E%3Csvg width='800px' height='800px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill-rule='evenodd' clip-rule='evenodd' d='M9.29289 1.29289C9.48043 1.10536 9.73478 1 10 1H18C19.6569 1 21 2.34315 21 4V7C21 7.55228 20.5523 8 20 8C19.4477 8 19 7.55228 19 7V4C19 3.44772 18.5523 3 18 3H11V8C11 8.55228 10.5523 9 10 9H5V20C5 20.5523 5.44772 21 6 21H11C11.5523 21 12 21.4477 12 22C12 22.5523 11.5523 23 11 23H6C4.34315 23 3 21.6569 3 20V8C3 7.73478 3.10536 7.48043 3.29289 7.29289L9.29289 1.29289ZM6.41421 7H9V4.41421L6.41421 7ZM18.25 20.75C18.25 21.4404 17.6904 22 17 22C16.3096 22 15.75 21.4404 15.75 20.75C15.75 20.0596 16.3096 19.5 17 19.5C17.6904 19.5 18.25 20.0596 18.25 20.75ZM15.1353 12.9643C15.3999 12.4596 16.0831 12 17 12C18.283 12 19 12.8345 19 13.5C19 14.1655 18.283 15 17 15C16.4477 15 16 15.4477 16 16V17C16 17.5523 16.4477 18 17 18C17.5523 18 18 17.5523 18 17V16.8866C19.6316 16.5135 21 15.2471 21 13.5C21 11.404 19.0307 10 17 10C15.4566 10 14.0252 10.7745 13.364 12.0357C13.1075 12.5248 13.2962 13.1292 13.7853 13.3857C14.2744 13.6421 14.8788 13.4535 15.1353 12.9643Z' fill='%23000000'/%3E%3C/svg%3E`,
		varIconName,
		`data:image/svg+xml,%3C%3Fxml%20version%3D%221.0%22%20encoding%3D%22iso-8859-1%22%3F%3E%0A%0A%3Csvg%20version%3D%221.1%22%20id%3D%22Capa_1%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20xmlns%3Axlink%3D%22http%3A%2F%2Fwww.w3.org%2F1999%2Fxlink%22%20x%3D%220px%22%20y%3D%220px%22%0A%09%20viewBox%3D%220%200%20512%20512%22%20style%3D%22enable-background%3Anew%200%200%20512%20512%3B%22%20xml%3Aspace%3D%22preserve%22%3E%0A%3Cpath%20style%3D%22fill%3A%23ECECF1%3B%22%20d%3D%22M421%2C0H91C49.6%2C0%2C16%2C33.6%2C16%2C75v362c0%2C41.4%2C33.6%2C75%2C75%2C75h330c41.4%2C0%2C75-33.6%2C75-75V75%0A%09C496%2C33.6%2C462.4%2C0%2C421%2C0z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23E2E2E7%3B%22%20d%3D%22M496%2C75v362c0%2C41.4-33.6%2C75-75%2C75H256V0h165C462.4%2C0%2C496%2C33.6%2C496%2C75z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%235A5A5A%3B%22%20d%3D%22M136%2C75v362c0%2C8.401-6.599%2C15-15%2C15s-15-6.599-15-15V75c0-8.401%2C6.599-15%2C15-15S136%2C66.599%2C136%2C75z%22%0A%09%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23444444%3B%22%20d%3D%22M136%2C75v362c0%2C8.401-6.599%2C15-15%2C15V60C129.401%2C60%2C136%2C66.599%2C136%2C75z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%235A5A5A%3B%22%20d%3D%22M226%2C75v362c0%2C8.401-6.599%2C15-15%2C15s-15-6.599-15-15V75c0-8.401%2C6.599-15%2C15-15S226%2C66.599%2C226%2C75z%22%0A%09%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23444444%3B%22%20d%3D%22M226%2C75v362c0%2C8.401-6.599%2C15-15%2C15V60C219.401%2C60%2C226%2C66.599%2C226%2C75z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%235A5A5A%3B%22%20d%3D%22M316%2C75v362c0%2C8.401-6.599%2C15-15%2C15s-15-6.599-15-15V75c0-8.401%2C6.599-15%2C15-15S316%2C66.599%2C316%2C75z%22%0A%09%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23444444%3B%22%20d%3D%22M316%2C75v362c0%2C8.401-6.599%2C15-15%2C15V60C309.401%2C60%2C316%2C66.599%2C316%2C75z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%235A5A5A%3B%22%20d%3D%22M406%2C75v362c0%2C8.401-6.599%2C15-15%2C15s-15-6.599-15-15V75c0-8.401%2C6.599-15%2C15-15S406%2C66.599%2C406%2C75z%22%0A%09%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23444444%3B%22%20d%3D%22M406%2C75v362c0%2C8.401-6.599%2C15-15%2C15V60C399.401%2C60%2C406%2C66.599%2C406%2C75z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF7816%3B%22%20d%3D%22M121%2C241c-24.901%2C0-45%2C21.099-45%2C46s20.099%2C45%2C45%2C45s45-20.099%2C45-45S145.901%2C241%2C121%2C241z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF4B00%3B%22%20d%3D%22M166%2C287c0%2C24.901-20.099%2C45-45%2C45v-91C145.901%2C241%2C166%2C262.099%2C166%2C287z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF7816%3B%22%20d%3D%22M391%2C90c-24.901%2C0-45%2C20.099-45%2C45s20.099%2C45%2C45%2C45s45-20.099%2C45-45S415.901%2C90%2C391%2C90z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF4B00%3B%22%20d%3D%22M436%2C135c0%2C24.901-20.099%2C45-45%2C45V90C415.901%2C90%2C436%2C110.099%2C436%2C135z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF7816%3B%22%20d%3D%22M301%2C332c-24.901%2C0-45%2C20.099-45%2C45s20.099%2C45%2C45%2C45s45-20.099%2C45-45S325.901%2C332%2C301%2C332z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF4B00%3B%22%20d%3D%22M346%2C377c0%2C24.901-20.099%2C45-45%2C45v-90C325.901%2C332%2C346%2C352.099%2C346%2C377z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF7816%3B%22%20d%3D%22M211%2C120c-24.901%2C0-45%2C20.099-45%2C45s20.099%2C45%2C45%2C45s45-20.099%2C45-45S235.901%2C120%2C211%2C120z%22%2F%3E%0A%3Cpath%20style%3D%22fill%3A%23FF4B00%3B%22%20d%3D%22M256%2C165c0%2C24.901-20.099%2C45-45%2C45v-90C235.901%2C120%2C256%2C140.099%2C256%2C165z%22%2F%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3Cg%3E%0A%3C%2Fg%3E%0A%3C%2Fsvg%3E%0A`,
		includedTaskfileIconName,
		`data:image/svg+xml,%3C%3Fxml version='1.0' encoding='utf-8'%3F%3E%3Csvg width='800px' height='800px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M3 5.25C3 4.00736 4.00736 3 5.25 3H18.75C19.9926 3 21 4.00736 21 5.25V12.0218C20.5368 11.7253 20.0335 11.4858 19.5 11.3135V5.25C19.5 4.83579 19.1642 4.5 18.75 4.5H5.25C4.83579 4.5 4.5 4.83579 4.5 5.25V18.75C4.5 19.1642 4.83579 19.5 5.25 19.5H11.3135C11.4858 20.0335 11.7253 20.5368 12.0218 21H5.25C4.00736 21 3 19.9926 3 18.75V5.25Z' fill='%23212121'/%3E%3Cpath d='M10.7803 7.71967C11.0732 8.01256 11.0732 8.48744 10.7803 8.78033L8.78033 10.7803C8.48744 11.0732 8.01256 11.0732 7.71967 10.7803L6.71967 9.78033C6.42678 9.48744 6.42678 9.01256 6.71967 8.71967C7.01256 8.42678 7.48744 8.42678 7.78033 8.71967L8.25 9.18934L9.71967 7.71967C10.0126 7.42678 10.4874 7.42678 10.7803 7.71967Z' fill='%23212121'/%3E%3Cpath d='M10.7803 13.2197C11.0732 13.5126 11.0732 13.9874 10.7803 14.2803L8.78033 16.2803C8.48744 16.5732 8.01256 16.5732 7.71967 16.2803L6.71967 15.2803C6.42678 14.9874 6.42678 14.5126 6.71967 14.2197C7.01256 13.9268 7.48744 13.9268 7.78033 14.2197L8.25 14.6893L9.71967 13.2197C10.0126 12.9268 10.4874 12.9268 10.7803 13.2197Z' fill='%23212121'/%3E%3Cpath d='M17.5 12C20.5376 12 23 14.4624 23 17.5C23 20.5376 20.5376 23 17.5 23C14.4624 23 12 20.5376 12 17.5C12 14.4624 14.4624 12 17.5 12ZM18.0011 20.5035L18.0006 18H20.503C20.7792 18 21.003 17.7762 21.003 17.5C21.003 17.2239 20.7792 17 20.503 17H18.0005L18 14.4993C18 14.2231 17.7761 13.9993 17.5 13.9993C17.2239 13.9993 17 14.2231 17 14.4993L17.0005 17H14.4961C14.22 17 13.9961 17.2239 13.9961 17.5C13.9961 17.7762 14.22 18 14.4961 18H17.0006L17.0011 20.5035C17.0011 20.7797 17.225 21.0035 17.5011 21.0035C17.7773 21.0035 18.0011 20.7797 18.0011 20.5035Z' fill='%23212121'/%3E%3Cpath d='M13.25 8.5C12.8358 8.5 12.5 8.83579 12.5 9.25C12.5 9.66421 12.8358 10 13.25 10H16.75C17.1642 10 17.5 9.66421 17.5 9.25C17.5 8.83579 17.1642 8.5 16.75 8.5H13.25Z' fill='%23212121'/%3E%3C/svg%3E`,
	)
	d2Writer.Write("vars", d2Vars)
	d2Writer.Write(uuid.NewString(), fmt.Sprintf(`Legend {
  **.style: {
    font-size: 30
    bold: true
  }
  near: top-center
  style.3d: true
  subLegend1: "" {
    style.opacity: 0
    grid-columns: 4
    grid-rows: 2
    icon1: Variable {
      shape: image
      icon: ${%s}
    }
    icon1Description: |md
      Variables are passed to tasks
    |
    icon2: External Task {
      shape: image
      icon: ${%s}
    }
    icon2Description: |md
      Tasks that can be called\
      directly by the Task CLI tool.
    |
    icon3: Internal Task {
      shape: image
      icon: ${%s}
    }
    icon3Description: |md
      Tasks that can NOT be called\
      directly by the Task CLI tool.
    |
    icon4: Unknown Task {
      shape: image
      icon: ${%s}
    }
    icon4Description: |md
      It is not possible to identify the origin of these\
      tasks as they are
      - a dynamically named task using template variable(s)\
        **or**
      - a task in another imported Taskfile
    |
    icon5: Included Taskfile {
      shape: image
      icon: ${%s}
    }
    icon5Description: |md
      Container for tasks that are included from other Taskfiles
    |
  }
  subLegend2: Silent Task {
    style: {
      fill: grey
    }
    description: |md
      Tasks that do NOT print their template resolution (**silent: true**).
      - This makes sure that **template resolution does not expose secret** variables
      - There could be other, less important reasons why a template resolution is not printed to the screen
    |
  }
}`, varIconName, externalTaskIconName, internalTaskIconName, unknownTaskIconName, includedTaskfileIconName))
	for _, include := range taskfile.GetIncludes() {
		includesToIncludedTasks[include] = make(map[string]struct{})
		// d2Writer.Write(fmt.Sprintf("'%s'", include), fmt.Sprintf("%s {}", include))
		d2Writer.Write(fmt.Sprintf("'%s'.icon", include), fmt.Sprintf("${%s}", includedTaskfileIconName))
	}

	// Tasks
	for _, taskName := range slices.Sorted(maps.Keys(taskfile.Tasks)) {
		task := taskfile.Tasks[taskName]
		// d2Writer.Write(fmt.Sprintf("'%s'", taskName), "{}")
		if task.Desc != "" || task.Summary != "" {
			markdownText := ""
			if task.Desc != "" {
				markdownText += fmt.Sprintf("## Description\n%s\n", task.Desc)
			}
			if task.Summary != "" {
				markdownText += fmt.Sprintf("## Summary\n%s\n", task.Summary)
			}
			d2Writer.Write(fmt.Sprintf("'%s'.Text", taskName), fmt.Sprintf("|md\n%s|", markdownText))
		}
		if task.Silent {
			d2Writer.Write(fmt.Sprintf("'%s'.style.fill", taskName), "grey")
		}
		var taskIcon string
		if task.Internal {
			taskIcon = internalTaskIconName
		} else {
			taskIcon = externalTaskIconName
		}
		d2Writer.Write(fmt.Sprintf("'%s'.icon", taskName), fmt.Sprintf("${%s}", taskIcon))

		// Required variables
		for _, requiredVar := range task.GetRequiredVars() {
			label := requiredVar.Name
			if len(requiredVar.Enum) != 0 {
				label = fmt.Sprintf("\"%s\\n[%s]\"", label, strings.Join(requiredVar.Enum, ", "))
			}
			d2Writer.Write(fmt.Sprintf("'%s'", requiredVar.Name), fmt.Sprintf("%s {shape: image; icon: ${%s}}", label, varIconName))
			d2Writer.Write(fmt.Sprintf("'%s' -> '%s'", requiredVar.Name, taskName), "required by")
		}

		// Dependency calls
		for _, depCall := range task.GetDepCalls() {
			EncapsulatePassedVars(d2Writer, taskName, &taskfile, depCall, "calls as dependency", "passed to {style {stroke-dash: 3; stroke: green}}")
		}

		// Internal task calls
		var callCount uint
		for _, taskCall := range task.GetCalls() {
			callCount++
			EncapsulatePassedVars(d2Writer, taskName, &taskfile, taskCall, fmt.Sprintf("calls (%v)", callCount), "passed to {style.stroke-dash: 3}")
		}
	}
	d2Writer.Write("(** -> **)[*].style",
		`{
  stroke-width: 4
  font-size: 25
  bold: true
}`)

	d2Writer.Write("(** -> **)[*]",
		`{
  &label: required by
  style {
    stroke: red
    stroke-dash: 3
  }
}`)

	d2Writer.Write("(** -> **)[*]",
		`{
  &label: calls as dependency
  style {
    stroke: green
  }
}`)

	d2Writer.Write("*", `{
  !&shape: image
  style.bold: true
  style.font-size: 30
}`)

	d2Writer.Write("**.icon",
		`{
  near: bottom-center
}`)

	d2Writer.Write("**",
		`{
  &shape: text
  style.font-size: 20
  style.bold: true
}`)
	return d2Writer.String(), nil
}

// ProcessIO processes the entire standard input as raw bytes using a handler function.
// The handler function processes the whole input and returns the transformed bytes or an error.
func ProcessIO(handler func([]byte) ([]byte, error)) error {
	// Read the entire input from stdin into a byte slice
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	// Pass the entire input to the handler
	processedData, err := handler(input)
	if err != nil {
		return fmt.Errorf("error processing input: %w", err)
	}
	// Write the processed data to stdout
	_, err = os.Stdout.Write(processedData)
	if err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}
	return nil
}
func EncapsulatePassedVars(d2Writer *D2Writer, taskName string, taskfile *Taskfile, taskCall TaskCall, firstConnectionValue, secondConnectionValue string) {
	// colons are for Taskfile namespaces for includes.
	// In the diagram it makes sense to place all included tasks into their parent Taskfile
	// representation to clearly show their relationship.
	calledD2TaskName := strings.ReplaceAll(taskCall.TaskName, ":", "'.'")
	if len(taskCall.Vars) == 0 {
		d2Writer.Write(fmt.Sprintf("'%s' -> '%s'", taskName, calledD2TaskName), firstConnectionValue)
	} else {
		passedVarsContainerUuid := uuid.NewString()
		d2Writer.Write(fmt.Sprintf("'%s' -> %s", taskName, passedVarsContainerUuid), firstConnectionValue)
		d2Writer.Write(fmt.Sprintf("%s -> '%s'", passedVarsContainerUuid, calledD2TaskName), secondConnectionValue)
		d2Writer.Write(passedVarsContainerUuid, "With {shape: parallelogram; style.stroke-dash: 3}")
		for _, passedVar := range taskCall.Vars {
			escaped := strings.NewReplacer("'", "\\'", "\"", "\\\"", "{", "\\{", "}", "\\}").Replace(fmt.Sprintf("%#v", passedVar.Value))
			d2Writer.Write(fmt.Sprintf("%s.'%s'", passedVarsContainerUuid, passedVar.Name), fmt.Sprintf("{shape: image; icon: ${%s}}", varIconName))
			valueUuid := uuid.NewString()
			d2Writer.Write(fmt.Sprintf("%s.%s", passedVarsContainerUuid, valueUuid), fmt.Sprintf("%v {shape: text}", escaped))
			d2Writer.Write(fmt.Sprintf("%s.'%s' -> %s.%s", passedVarsContainerUuid, passedVar.Name, passedVarsContainerUuid, valueUuid), "set to")
		}
	}
	if strings.Contains(taskCall.TaskName, ":") {
		taskNameChunks := strings.SplitN(taskCall.TaskName, ":", 2)
		includedTasks := includesToIncludedTasks[taskNameChunks[0]]
		if _, alreadyHasIcon := includedTasks[taskNameChunks[1]]; !alreadyHasIcon {
			includedTasks[taskNameChunks[1]] = struct{}{}
			d2Writer.Write(fmt.Sprintf("'%s'.icon", calledD2TaskName), fmt.Sprintf("${%s}", unknownTaskIconName))
		}
	} else if _, taskExists := taskfile.Tasks[calledD2TaskName]; !taskExists {
		d2Writer.Write(fmt.Sprintf("'%s'.icon", calledD2TaskName), fmt.Sprintf("${%s}", unknownTaskIconName))
	}
}
