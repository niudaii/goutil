package v1

const (
	ConfigFile    = "config.yaml"
	OperationFile = "OPERATION"
)

const (
	TaskStarted      = "Task Started"
	TaskCancelled    = "Task Cancelled"
	TaskTimeoutEnded = "Task Timeout Ended"
	TaskFinished     = "Task Finished"
	TaskProgress     = "Task Progress: %v/%v"

	RunStarted  = "Run Started => %v"
	RunCount    = "Run Count => %v"
	RunResult   = "Run Result %v => %v"
	RunFinished = "Run Finished spent=%v"
	RunFailed   = "Run Failed %v"
	RunTimeout  = "Run Timeout maxRuntime=%v"

	ProgramStarted = "Program Started"
	ProgramExited  = "Program Exited"
	ProgramError   = "Program err: %v"

	TargetCount = "Target Count => %v"

	EmptyResult = "Result is Empty"
	SaveResult  = "Result Saved => %v"

	ScanConfiguration   = "Scan Config =>\n%v"
	NewRunnerError      = "NewRunner err: %v"
	InitializationError = "Initialization Failed: %v"

	UnsupportedProtocol = "Unsupported Protocol: %v"
)
