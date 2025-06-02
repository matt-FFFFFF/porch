package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "foreachdirectory"

// init registers the foreachdirectory command type.
func init() {
	commandregistry.Register(commandType, &Commander{})
}
