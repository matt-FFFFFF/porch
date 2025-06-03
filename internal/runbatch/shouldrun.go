package runbatch

type ShouldRunAction int

const (
	ShouldRunActionRun   ShouldRunAction = iota // Run the command
	ShouldRunActionSkip                         // Skip the command
	ShouldRunActionError                        // An error occurred, do not run the command
)
