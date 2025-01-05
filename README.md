
# Taskfile2D2
`taskfile2d2` is a command-line tool that converts a [Taskfile](https://taskfile.dev/#/) YAML file into a [D2 diagram](https://d2lang.com/), a declarative language for visualizing data structures. It allows you to generate visual representations of task interactions.

The resulting diagram can be opened with [D2](https://github.com/terrastruct/d2) or by using [D2 Playground](https://play.d2lang.com/).
I recommend using the **ELK** layout engine, as it does a much better rendering job than the default **Dagre**. To do this in D2 CLI, set the "--layout" flag to "elk".

## Value Proposition
- Easier onboarding. Task itself already makes onboarding easy, but a large Taskfile can be intimidating for newcomers. Diagramming is a familiar language to everyone.
- Efficient inter-department communication. During meetings, it is easier to keep brainstorming when all parties are looking at the same diagram and not a large YAML file.
- Safer Taskfile refactoring. A Taskfile can outgrow its original design, in which case refactoring is necessary to streamline the workflows. But it gets riskier the more people depend on the Taskfile's workflow, so all Taskfile maintainers must be on the same page (diagram) before making these changes.



## Features
- Converts Taskfiles (version 3) into D2 diagrams.
- Visualizes tasks, dependencies, and variable requirements in an organized diagram.
- Supports input via file, standard input, or URL.
- Output diagrams in `.d2` format.

## Installation
Compiled binaries are available on the [Releases](https://github.com/NorbertHauriel/taskfile2d2/releases) page.

## Usage
The `taskfile2d2` command can be used in several ways:

### Passing Input as Argument
- Generate a D2 diagram from a Taskfile and save to the default output:

  ```bash
  taskfile2d2 Taskfile.yml
  ```

  This creates a file named `Taskfile.yml.d2`.

- Specify a custom output file:

  ```bash
  taskfile2d2 Taskfile.yml output.d2
  ```

### Using Standard Input
- Generate a diagram by piping a Taskfile:

  ```bash
  cat Taskfile.yml | taskfile2d2 > output.d2
  ```

- Use input redirection:

  ```bash
  taskfile2d2 < Taskfile.yml > output.d2
  ```

- Fetch a Taskfile from a remote source:

  ```bash
  curl -s http://example.com/Taskfile.yml | taskfile2d2 > output.d2
  ```

- Print D2 content to the terminal:

  ```bash
  taskfile2d2 < Taskfile.yml
  ```
## Upcoming Features
Although `taskfile2d2` is fully functional, imrovements on the **diagram** and **customizability** may come in the future.

## Special Thanks
- [Task](https://github.com/go-task) for creating a 21st-century replacement for GNU Make.
- [Terrastruct](https://github.com/terrastruct) for creating D2, a standard in diagramming as groundbreaking as Markdown for documentation.