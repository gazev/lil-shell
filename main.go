package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
)

var lr int = 0

var builtins = map[string]struct{}{
	"exit": {},
	"type": {},
	"echo": {},
	"pwd":  {},
	"cd":   {},
	"lr":   {}, // lastresult like $?
}

func main() {
	for {
		fmt.Fprintf(os.Stdout, "[%s@%s %s]$ ", getUser(), getHost(), getUnexpandedCwd())
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				os.Exit(0)
			}
			fmt.Printf("failed reading from stdin: %s\n", err)
			continue
		}

		var tokens []string = ParseInput(input)
		if tokens == nil { // empty line
			continue
		}
		var commands [][]string = GetPipeSeparatedCommands(tokens)

		if len(commands) > 1 {
			if err := runPipedCommands(commands...); err != nil {
				lr = 1
				fmt.Printf("%s\n", err)
			} else {
				lr = 0
			}
			continue
		}

		var cmd string = commands[0][0]
		if cmd == "" {
			continue
		}
		var args []string
		var argc int
		if len(tokens) > 1 {
			args = tokens[1:]
			argc = len(args)
		}

		switch cmd {
		case "exit":
			if argc < 1 {
				os.Exit(0)
			}
			ret, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("%s: invalid argument \"%s\"\n", cmd, args[0])
				lr = 1
				break
			}
			os.Exit(ret)
		case "echo":
			aux := make([]any, argc) // fighting the type checker
			for i, v := range args {
				aux[i] = v
			}
			fmt.Println(aux...)
			lr = 0
		case "type":
			if argc < 1 {
				fmt.Printf("%s: missing argument\n", cmd)
				lr = 1
				break
			}
			if _, exists := builtins[args[0]]; exists {
				fmt.Printf("%s is a shell builtin\n", args[0])
				lr = 0
				break
			}
			path := findInPath(args[0])
			if path == "" {
				fmt.Printf("%s: not found\n", args[0])
				lr = 1
				break
			}
			fmt.Printf("%s is %s\n", args[0], path)
			lr = 0
		case "pwd":
			dir, err := os.Getwd()
			if err != nil {
				fmt.Printf("%s failed: %s", cmd, err)
				lr = 1
				break
			}
			fmt.Printf("%s\n", dir)
			lr = 0
		case "cd":
			if argc < 1 {
				args = append(args, "~") // defaults to HOME
			}
			dirPath := expandHome(args[0])
			if dirPath == "" {
				lr = 1
				fmt.Printf("%s: Couldn't expand ~, $HOME not set?\n", cmd)
				break
			}
			if err := os.Chdir(dirPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					fmt.Printf("%s: %s: No such file or directory\n", cmd, dirPath)
				} else {
					var fsErr *fs.PathError
					if errors.As(err, &fsErr) {
						fmt.Printf("%s: %s: Not a directory\n", cmd, dirPath)
					} else {
						fmt.Printf("%s: %s: Couldn't complete command\n", cmd, dirPath)
					}
				}
				lr = 1
				break
			}
			lr = 0
		case "lr":
			fmt.Printf("%d\n", lr)
		default:
			path := findInPath(cmd)
			if path == "" {
				fmt.Printf("%s: command not found\n", cmd)
				lr = 1
				break
			}
			cmdT := exec.Command(cmd, args...)
			cmdT, err = startCommand(cmdT, os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				fmt.Printf("%s: %s\n", cmd, err)
				lr = 1
				break
			}
			if err := cmdT.Wait(); err != nil {
				var exitErr exec.ExitError
				if !errors.Is(err, &exitErr) {
					fmt.Printf("%s: failed: %s\n", cmd, err)
					lr = 1
					break
				}
			}
			lr = cmdT.ProcessState.ExitCode()
		}
	}
}

func startCommand(cmd *exec.Cmd, in io.Reader, out io.Writer, err io.Writer) (*exec.Cmd, error) {
	cmd.Stdout = out
	cmd.Stdin = in
	cmd.Stderr = err
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed starting command: %w", err)
	}
	return cmd, nil
}

func runPipedCommands(commands ...[]string) error {
	cmd := commands[0][0]
	path := findInPath(cmd)
	if path == "" {
		return fmt.Errorf("command \"%s\" not found", cmd)
	}
	var args []string
	if len(commands[0]) > 1 {
		args = commands[0][1:]
	}

	// First command inherits stdin for input
	cmdT := exec.Command(cmd, args...)
	previousStdoutPipe, err := cmdT.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed creating pipe for %s stdout: %w", cmd, err)
	}

	if err := cmdT.Start(); err != nil {
		return fmt.Errorf("failed starting command %s: %w", cmd, err)
	}

	var lastcmd string // keep last iteration cmd string
	for i := 1; i < len(commands); i++ {
		cmd = commands[i][0]
		lastcmd = cmd
		path := findInPath(cmd)
		if path == "" {
			return fmt.Errorf("command \"%s\" not found", cmd)
		}
		var args []string
		if len(commands[i]) > 1 {
			args = commands[i][1:]
		}

		nextCmd := exec.Command(cmd, args...)
		nextCmd.Stdin = previousStdoutPipe
		if i < len(commands)-1 { // last command inherits stdout
			previousStdoutPipe, err = nextCmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("failed creating pipe for %s stdout: %w", cmd, err)
			}
			nextCmd.Stderr = os.Stderr
		} else {
			nextCmd.Stdout = os.Stdout
			nextCmd.Stderr = os.Stderr
		}

		if err := nextCmd.Start(); err != nil {
			return fmt.Errorf("failed starting command %s: %w", cmd, err)
		}

		if err := cmdT.Wait(); err != nil {
			return fmt.Errorf("%s: failed: %s", cmd, err)
		}

		cmdT = nextCmd
	}

	if err := cmdT.Wait(); err != nil {
		return fmt.Errorf("%s: failed: %s", lastcmd, err)
	}

	return nil
}
