package v1

const (
	LoginSuccess       = "登录成功"
	LoginErrorPass     = "登陆失败，用户名或密码不正确"
	LoginErrorCode     = "登陆失败，验证码不正确"
	UsernameDuplicated = "账户重复"
	Logout             = "注销成功"

	Success = "成功"
	Fail    = "失败"

	FindSuccess = "查询成功"
	FindFail    = "查询失败"

	UpdateSuccess = "更新成功"
	UpdateFail    = "更新失败"

	CreateSuccess = "创建成功"
	CreateFail    = "创建失败"

	DeleteSuccess = "删除成功"
	DeleteFail    = "删除失败"

	EngineClose     = "引擎当前为关闭状态"
	EngineCloseWarn = "引擎当前为关闭状态，无法执行任务"
	EngineSwitch    = "引擎状态切换为 %v"

	TaskCreateSuccess = "任务创建成功"
	TaskCreateFail    = "任务创建失败"
	TaskStartSuccess  = "任务启动成功"
	TaskStartFail     = "任务启动失败"
	TaskCancelSuccess = "任务取消成功"
	TaskCancelFail    = "任务取消失败: %v"
)
