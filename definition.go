package main

import (
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	//версия сервера или ноды, пока не используется
	REVISIT_VERSION = "0.5"

	//общие константы
	CODE_LENGTH     = 64 //длина code
	PASSWORD_LENGTH = 14
	FILE_PROFILES   = "profiles.list"
	FILE_OPTIONS    = "options.cfg"
	FILE_COUNTERS   = "counters.json"
	FILE_VNCLIST    = "vnc.list"
	LOG_NAME        = "log.txt"
	MAX_LEN_ID_LOG  = 6
	MAX_LEN_ID_NODE = 8
	LEN_SALT        = 16

	//константы ожидания
	WAIT_COUNT         = 30
	WAIT_IDLE          = 500
	WAIT_AFTER_CONNECT = 250
	WAIT_HELPER_CYCLE  = 5
	WAIT_PING          = 10
	WAIT_IDLE_AGENT    = 2

	//виды сообщений логов
	MESS_ERROR  = 1
	MESS_INFO   = 2
	MESS_DETAIL = 3
	MESS_FULL   = 4

	//виды сообщений
	TMESS_DEAUTH          = 0  //деаутентификация()
	TMESS_VERSION         = 1  //запрос версии
	TMESS_AUTH            = 2  //аутентификация(генерация pid)
	TMESS_LOGIN           = 3  //вход в профиль
	TMESS_NOTIFICATION    = 4  //сообщение клиент
	TMESS_REQUEST         = 5  //запрос на подключение
	TMESS_CONNECT         = 6  //запрашиваем подключение у клиента
	TMESS_DISCONNECT      = 7  //сообщаем об отключении клиенту
	TMESS_REG             = 8  //регистрация профиля
	TMESS_CONTACT         = 9  //создание, редактирование, удаление
	TMESS_CONTACTS        = 10 //запрос списка контактов
	TMESS_LOGOUT          = 11 //выход из профиля
	TMESS_CONNECT_CONTACT = 12 //запрос подключения к конакту из профиля
	TMESS_STATUSES        = 13 //запрос всех статусов
	TMESS_STATUS          = 14 //запрос статуса
	TMESS_INFO_CONTACT    = 15 //запрос информации о клиенте
	TMESS_INFO_ANSWER     = 16 //ответ на запрос информации
	TMESS_MANAGE          = 17 //запрос на управление(перезагрузка, обновление, переустановка)
	TMESS_PING            = 18 //проверка состояния подключения
	TMESS_CONTACT_REVERSE = 19 //добавление себя в чужой профиль

	TMESS_AGENT_DEAUTH      = 0
	TMESS_AGENT_AUTH        = 1
	TMESS_AGENT_ANSWER      = 2
	TMESS_AGENT_ADD_CODE    = 3
	TMESS_AGENT_DEL_CODE    = 4
	TMESS_AGENT_NEW_CONNECT = 5
	TMESS_AGENT_DEL_CONNECT = 6
	TMESS_AGENT_ADD_BYTES   = 7

	REGULAR = 0
	MASTER  = 1
	NODE    = 2
)

var (

	//опции по-умолчанию
	options = Options{
		MainServerPort: "65471",
		DataServerPort: "65475",
		HttpServerPort: "8090",
		SizeBuff:       16000,
		AdminLogin:     "admin",
		AdminPass:      "admin",
		Mode:           REGULAR,
		FDebug:         true,
		MasterServer:   "data.rvisit.net",
		MasterPort:     "65470",
		MasterPassword: "master",
	}

	//считаем всякую бесполезную информацию или нет
	counterData struct {
		currentPos time.Time

		CounterBytes       [24]uint64
		CounterConnections [24]uint64
		CounterClients     [24]uint64

		CounterDayWeekBytes       [7]uint64
		CounterDayWeekConnections [7]uint64
		CounterDayWeekClients     [7]uint64

		CounterDayBytes       [31]uint64
		CounterDayConnections [31]uint64
		CounterDayClients     [31]uint64

		CounterDayYearBytes       [365]uint64
		CounterDayYearConnections [365]uint64
		CounterDayYearClients     [365]uint64

		CounterMonthBytes       [12]uint64
		CounterMonthConnections [12]uint64
		CounterMonthClients     [12]uint64

		mutex sync.Mutex
	}

	//меню веб интерфейса админки
	menuAdmin = []itemMenu{
		{"Логи", "/admin/logs"},
		{"Настройки", "/admin/options"},
		{"Ресурсы", "/admin/resources"},
		{"Статистика", "/admin/statistics"},
		{"reVisit", "/resource/reVisit.exe"}}

	//меню веб интерфейса профиля
	menuProfile = []itemMenu{
		{"Профиль", "/profile/my"},
		{"reVisit", "/resource/reVisit.exe"}}

	//максимальный уровень логов
	typeLog = MESS_FULL

	//файл для хранения лога
	logFile *os.File

	//карта подключенных клиентов
	clients sync.Map

	//карта каналов для передачи данных
	channels sync.Map

	//карта учеток
	profiles sync.Map

	//агенты обработки данных
	nodes sync.Map

	//сокет до мастера
	master *net.Conn

	//текстовая расшифровка сообщений для логов
	messLogText = []string{
		"BLANK",
		"ERROR",
		"INFO",
		"DETAIL",
		"FULL"}

	//функции для обработки сообщений
	processing = []ProcessingMessage{
		{TMESS_DEAUTH, nil},
		{TMESS_VERSION, processVersion},
		{TMESS_AUTH, processAuth},
		{TMESS_LOGIN, processLogin},
		{TMESS_NOTIFICATION, processNotification},
		{TMESS_REQUEST, processConnect},
		{TMESS_CONNECT, nil},
		{TMESS_DISCONNECT, processDisconnect},
		{TMESS_REG, processReg},
		{TMESS_CONTACT, processContact},
		{TMESS_CONTACTS, processContacts},
		{TMESS_LOGOUT, processLogout},
		{TMESS_CONNECT_CONTACT, processConnectContact},
		{TMESS_STATUSES, processStatuses},
		{TMESS_STATUS, processStatus},
		{TMESS_INFO_CONTACT, processInfoContact},
		{TMESS_INFO_ANSWER, processInfoAnswer},
		{TMESS_MANAGE, processManage},
		{TMESS_PING, processPing},
		{TMESS_CONTACT_REVERSE, processContactReverse}}

	processingAgent = []ProcessingAgent{
		{TMESS_AGENT_DEAUTH, nil},
		{TMESS_AGENT_AUTH, processAgentAuth},
		{TMESS_AGENT_ANSWER, processAgentAnswer},
		{TMESS_AGENT_ADD_CODE, processAgentAddCode},
		{TMESS_AGENT_DEL_CODE, processAgentDelCode},
		{TMESS_AGENT_NEW_CONNECT, processAgentNewConnect},
		{TMESS_AGENT_DEL_CONNECT, processAgentDelConnect},
		{TMESS_AGENT_ADD_BYTES, processAgentAddBytes}}

	//функции для обработки web api
	processingWeb = []ProcessingWeb{
		{"defaultvnc", processApiDefaultVnc},
		{"listvnc", processApiListVnc},
		{"getlog", processApiGetLog},
		{"clearlog", processApiClearLog},
		{"profile_save", processApiProfileSave},
		{"profile_get", processApiProfileGet},
		{"save_options", processApiSaveOptions},
		{"options_save", processApiOptionsSave},
		{"reload", processApiReload},
		{"options_get", processApiOptionsGet}}

	//список доступных vnc клиентов и выбранный по-умолчанию
	defaultVnc = 0
	arrayVnc   []VNC
)

//double pointer
type dConn struct {
	pointer [2]*net.Conn
	flag    [2]bool
	node    [2]*Node
	mutex   sync.Mutex
}

//информацияя о ноде
type Node struct {
	Id   string
	Name string
	Ip   string
	Conn *net.Conn
}

//обработчик для веб запроса
type ProcessingWeb struct {
	Make       string
	Processing func(w http.ResponseWriter, r *http.Request)
}

//обработчик для запросов агенту
type ProcessingAgent struct {
	TMessage   int
	Processing func(message Message, conn *net.Conn, curNode *Node, id string)
}

//обработчик для сообщений
type ProcessingMessage struct {
	TMessage   int
	Processing func(message Message, conn *net.Conn, curClient *Client, id string)
}

//тип для сообщения
type Message struct {
	TMessage int
	Messages []string
}

//сохраняемые опции
type Options struct {
	//настройки smtp сервера
	ServerSMTP string
	PortSMTP   string
	LoginSMTP  string
	PassSMTP   string

	//реквизиты сервера
	MainServerPort string

	//реквизиты сервер
	DataServerPort string

	//реквизиты веб сервера
	HttpServerPort string

	//размер буфера для операций с сокетами
	SizeBuff int

	//учетка для админ панели
	AdminLogin string
	AdminPass  string

	//режим работы экземпляра сервера
	Mode int

	//мастер сервер, если он нужен
	MasterServer   string
	MasterPort     string
	MasterPassword string

	//очевидно что флаг для отладки
	FDebug bool
}

//информация о внц и основные команды для управления им
type VNC struct {
	FileServer string
	FileClient string

	//это команды используем для старта под админскими правами(обычно это создание сервиса)
	CmdStartServer   string
	CmdStopServer    string
	CmdInstallServer string
	CmdRemoveServer  string
	CmdConfigServer  string
	CmdManageServer  string

	//это комнады используем для старта без админских прав
	CmdStartServerUser   string
	CmdStopServerUser    string
	CmdInstallServerUser string
	CmdRemoveServerUser  string
	CmdConfigServerUser  string
	CmdManageServerUser  string

	//комнды для vnc клиента
	CmdStartClient   string
	CmdStopClient    string
	CmdInstallClient string
	CmdRemoveClient  string
	CmdConfigClient  string
	CmdManageClient  string

	PortServerVNC string
	Link          string
	Name          string
	Version       string
	Description   string
}

//меню для веба
type itemMenu struct {
	Capt string
	Link string
}

//тип для клиента
type Client struct {
	Serial  string
	Pid     string
	Pass    string
	Version string
	Salt    string //for password
	Profile *Profile

	Conn *net.Conn
	Code string //for connection

	profiles sync.Map //профили которые содержат этого клиента в контактах(используем для отправки им информации о своем статусе)
}

//тип для профиля
type Profile struct {
	Email string
	Pass  string

	Contacts *Contact
	mutex    sync.Mutex

	clients sync.Map //клиенты которые авторизовались в этот профиль(используем для отправки им информации о статусе или изменений контактов)

	//всякая информация
	Capt string
	Tel  string
	Logo string
}

//тип для контакта
type Contact struct {
	Id      int
	Caption string
	Type    string //cont - контакт, fold - папка
	Pid     string
	Digest  string //но тут digest
	Salt    string

	Inner *Contact
	Next  *Contact
}
