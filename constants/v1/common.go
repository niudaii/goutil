package v1

const (
	ConfigFile    = "config.yaml"
	OperationFile = "OPERATION"
)

const (
	TaskStart    = "任务开始"
	TaskCanceled = "任务已取消"
	TaskTimeout  = "任务超时结束"
	TaskFinished = "任务完成"
	TaskProgress = "任务进度 => %v/%v"

	RunStart  = "开始运行 => %v"
	RunCount  = "运行数量 %v"
	RunResult = "运行结果 %v => %v"
	RunEnd    = "运行结束 spent=%v"
	RunError  = "运行失败 %v"

	RunTimeout = "运行超出最大执行时间 maxRuntime=%v"
)

// cmd
const (
	ProgramStart = "程序启动"
	ProgramExit  = "程序退出\n"
	ProgramError = "程序报错 %v"

	FileNotExist = "文件不存在 %v"
	TargetCount  = "目标数量 %v"

	EmptyResult = "结果为空"
	SaveResult  = "结果保存 => %v"

	ScanConfig = "扫描配置 =>\n%v"
	InitError  = "初始化失败 %v"

	UnSupportProtocol = "不支持的协议: %v"
)
