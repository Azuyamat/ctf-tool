package commands

import (
	"fmt"
	"io"

	"github.com/azuyamat/gear/command"
)

const posixWrapper = `ctf() {
    if [ "$1" = cd ]; then
        shift
        local dir
        dir=$(CTF_CD=1 command ctf cd "$@") && builtin cd -- "$dir"
    else
        command ctf "$@"
    fi
}
`

const fishWrapper = `function ctf
    if test "$argv[1]" = cd
        set -l dir (CTF_CD=1 command ctf cd $argv[2..-1]); and builtin cd -- $dir
    else
        command ctf $argv
    end
end
`

func cmdInit(root *command.RootCommand) command.Command {
	return command.NewExecutableCommand("init", "Print shell setup for ctf cd and completion").
		Group("Setup").
		Args(command.NewStringArg("shell", "Shell to set up").Choices("bash", "zsh", "fish")).
		Example(
			"# ~/.zshrc or ~/.bashrc\neval \"$(ctf init zsh)\"",
			"# ~/.config/fish/config.fish\nctf init fish | source",
		).
		Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
			shell := command.ArgValue[string](args, "shell")
			return writeInit(ctx.Out(), root, shell)
		})
}

func writeInit(w io.Writer, root *command.RootCommand, shell string) error {
	wrapper := posixWrapper
	if shell == "fish" {
		wrapper = fishWrapper
	}
	if _, err := fmt.Fprintln(w, wrapper); err != nil {
		return err
	}
	return root.WriteCompletion(w, shell)
}
