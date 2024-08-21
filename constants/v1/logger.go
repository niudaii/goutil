package v1

// Logger Levels
const (
	DebugLevel  = "debug"
	InfoLevel   = "info"
	WarnLevel   = "warn"
	ErrorLevel  = "error"
	SlientLevel = "slient"
)

// Logger Keys
const (
	TimeKey  = "time"
	StackKey = "stack"
	ErrorKey = "error"

	RequestKey = "request"
	TopicKey   = "topicName"
)

// Logger Names
const (
	GinLogger  = "[gin]"
	GormLogger = "[gorm]"

	HTTPClientLogger = "[http_client]"

	KafkaProducerLogger = "[kafka_producer]"
	KafkaCustomerLogger = "[kafka_consumer]"

	NacosConfig = "[nacos_config]"
)

// Debug
const (
	Request  = "request => [%v]\n%v"
	Response = "response =>\n%v"

	ClientRequest  = "client => [%v]\n%v"
	ClientResponse = "response => [%v]\n%v"
)

// Info
const (
	HTTPRouter = "%v %v --> %v"
	HTTPServe  = "HTTP Server Will Listening at %v"

	KafkaRouter = "%v --> %v"
	KafkaServe  = "KAFKA Consumer Will Listening at %v"

	CreateTopc = "create topic: %v"

	InitDataExist   = "the initial data for the table %v already exists"
	InitDataError   = "initialize table(%v) data failed: %v"
	InitDataSuccess = "initialize table(%v) data success"

	SignalExit = "signal received: %v, program exit\n\n"

	InitEngineSuccess = "init engine success =>\n%v"

	ScanResult = "%v => %v"
	ScanInput  = "input =>\n%v"
)

// Error
const (
	HTTPServeError = "HTTP Server ListenAndServe err: %v"

	RequestError = "request err: %v"

	ProducerError = "produce err: %v"
	ProduceOk     = "produce ok => partition=%v offset=%v msg.bytes=%v"

	InitKafkaConsumerError  = "init kafka consumer err: %v"
	InitKafkaProducerError  = "init kafka producer err: %v"
	CloseKafkaConsumerError = "consumer close err: %v"
	GetPartitionsError      = "get partitions err: %v"
	ConsumePartitionError   = "consume partition [%v] err: %v"

	RecoverCommon      = "[recover from panic] error => %v\n"
	RecoverWithStack   = RecoverCommon + "stack =>\n%v\n"
	RecoverWithRequest = RecoverCommon + "request =>\n%v\n"
	RecoverWithAll     = RecoverCommon + "request =>\n%v\nstack =>\n%v\n"

	EmptyDBNameError     = "dbName can't be empty"
	UnSupportDBTypeError = "dbType only support mysql && pgsql"

	ParseConfigError = "error parse config %v err: %v\n"
	InitLoggerError  = "error Init logger err: %v\n"

	ConnDBError        = "conn to the db err: %v"
	RegisterTableError = "register table err: %v"

	InitEngineError = "init engine err: %v"

	ParseRequestError = "parse request err: %v"

	InvalidOperation = "invalid operation err: %v"

	InitUsecaseError = "init usecase err: %v"
)

const (
	TaskLimitExceeded = "task limit exceeded. limit: %v, current: %v"
	NoPluginsLoaded   = "no plugin loaded"
	CheckAliveError   = "check alive err: %v"
	FileNotExist      = "file %v does not exist"
)
