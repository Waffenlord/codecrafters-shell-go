package main

import (
	"io"
	"os"
	"os/exec"

	"golang.org/x/term"
)

type pipelineCommand interface {
	start(termState *term.State) error
	wait() error
	setStdin(io.Reader)
	setStdout(io.Writer)
	setStderr(io.Writer)
	readsStdin() bool
	ownsStdout() bool
}

type externalCmd struct {
	cmd    *exec.Cmd
	rStdin bool
}

func (e *externalCmd) setStdin(r io.Reader)  { e.cmd.Stdin = r }
func (e *externalCmd) setStdout(w io.Writer) { e.cmd.Stdout = w }
func (e *externalCmd) setStderr(w io.Writer) { e.cmd.Stderr = w }

func (e *externalCmd) start(_ *term.State) error { return e.cmd.Start() }
func (e *externalCmd) wait() error               { return e.cmd.Wait() }
func (e *externalCmd) readsStdin() bool          { return e.rStdin }
func (e *externalCmd) ownsStdout() bool          { return false }

func newExternalCmd(name string, args []string) *externalCmd {
	return &externalCmd{
		cmd:    exec.Command(name, args...),
		rStdin: true,
	}
}

type builtinCmd struct {
	fn     builtin
	args   []string
	in     io.Reader
	out    io.Writer
	errOut io.Writer
	done   chan error
	rStdin bool
}

func (b *builtinCmd) setStdin(r io.Reader)  { b.in = r }
func (b *builtinCmd) setStdout(w io.Writer) { b.out = w }
func (b *builtinCmd) setStderr(w io.Writer) { b.errOut = w }

func (b *builtinCmd) start(termState *term.State) error {
	go func() {
		err := b.fn(b.in, b.out, b.args, termState)
		if c, ok := b.out.(io.Closer); ok && c != os.Stdout && c != os.Stderr {
			c.Close()
		}
		b.done <- err
	}()
	return nil
}

func (b *builtinCmd) wait() error {
	return <-b.done
}
func (b *builtinCmd) readsStdin() bool { return b.rStdin }
func (b *builtinCmd) ownsStdout() bool { return true }

func newBuiltinCmd(fn builtin, args []string) *builtinCmd {
	return &builtinCmd{
		fn:   fn,
		args: args,
		done: make(chan error, 1),
	}
}

func processPipeline(commands []commandReceived, menu builtInMenu, termOldState *term.State) error {
	pipelineCommands := make([]pipelineCommand, len(commands))

	for i, c := range commands {
		isBuiltIn := menu.isBuiltIn(c.command)
		if isBuiltIn {
			cmd := menu.commands[c.command]
			pipelineCommands[i] = newBuiltinCmd(cmd, c.params)
		} else {
			paramsWithoutSpaces := filterSpacesFromParams(c.params)
			pipelineCommands[i] = newExternalCmd(c.command, paramsWithoutSpaces)
		}
	}

	pipes := make([][2]*os.File, len(pipelineCommands)-1)

	for i := 0; i < len(pipes); i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		pipes[i] = [2]*os.File{r, w}
	}

	for i, pc := range pipelineCommands {
		if i == 0 {
			pc.setStdin(os.Stdin)
		} else {
			pc.setStdin(pipes[i-1][0])
		}

		if i == len(pipelineCommands)-1 {
			pc.setStdout(os.Stdout)
		} else {
			pc.setStdout(pipes[i][1])
		}

		pc.setStderr(os.Stderr)

		if err := pc.start(termOldState); err != nil {
			return err
		}
	}

	for _, p := range pipes {
		p[0].Close()
		p[1].Close()
	}

	for _, pc := range pipelineCommands {
		pc.wait()
	}
	return nil
}
