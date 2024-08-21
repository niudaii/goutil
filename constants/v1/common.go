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

	RunStarted  = "Run Started: %v"
	RunCount    = "Run Count: %v"
	RunResult   = "Run Result: %v => %v"
	RunFinished = "Run Finished: Duration=%v"
	RunFailed   = "Run Failed: %v"
	RunTimeout  = "Run Timeout: MaxRuntime=%v"

	ProgramStarted = "Program Started"
	ProgramExited  = "Program Exited"
	ProgramError   = "Program Error: %v"

	FileNotExist = "Error: File does not exist: %v"
	TargetCount  = "Target Count: %v"

	EmptyResult = "Result is Empty"
	SaveResult  = "Result Saved: %v"

	ScanConfiguration   = "Scan Configuration:\n%v"
	NewRunnerError      = "NewRunner Error: %v"
	InitializationError = "Initialization Failed: %v"

	UnsupportedProtocol = "Unsupported Protocol: %v"
)
