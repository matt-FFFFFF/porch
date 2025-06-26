package hcl

import (
	"errors"
	"fmt"
	"github.com/Azure/golden"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/peterh/liner"
)

func EnterDebugMode(config PorchConfig) {
	line := liner.NewLiner()
	defer func() {
		_ = line.Close()
	}()

	line.SetCtrlCAborts(true)
	fmt.Println("Entering debugging mode, press `quit` or `exit` or Ctrl+C to quit.")

	for {
		if input, err := line.Prompt("debug> "); err == nil {
			if input == "quit" || input == "exit" {
				return
			}
			line.AppendHistory(input)
			expression, diag := hclsyntax.ParseExpression([]byte(input), "repl.hcl", hcl.InitialPos)
			if diag.HasErrors() {
				fmt.Printf("%s\n", diag.Error())
				continue
			}
			value, diag := expression.Value(config.EvalContext())
			if diag.HasErrors() {
				fmt.Printf("%s\n", diag.Error())
				continue
			}
			fmt.Println(golden.CtyValueToString(value))
		} else if errors.Is(err, liner.ErrPromptAborted) {
			fmt.Println("Aborted")
			break
		} else {
			fmt.Println("Error reading line: ", err)
			break
		}
	}

	return
}
