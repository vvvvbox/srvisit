package main

import (
	"net"
	"sync"
	"os"
	"net/http"
	"time"
)

const(
	REVISIT_VERSION = "0.5"

	//общие константы
	CODE_LENGTH = 64 //длина code
	PASSWORD_LENGTH = 14
	FILE_PROFILES = "profiles.list"
	FILE_OPTIONS = "options.cfg"
	FILE_COUNTERS = "counters.json"
	FILE_VNCLIST = "vnc.list"
	LOG_NAME = "log.txt"
	PORT_FINDER_NEIGHBOURS = 1231
	MAX_LEN_ID_NEIGHBOUR = 8
	LEN_SALT = 16

	//константы ожидания
	WAIT_COUNT = 15
	WAIT_IDLE = 500
	WAIT_AFTER_CONNECT = 250
	WAIT_HELPER_CYCLE = 5
	WAIT_PING = 10
	WAIT_IDLE_FINDER = 5
	WAIT_IDLE_CLEANER = 11

	//виды сообщений логов
	MESS_ERROR  = 1
	MESS_INFO   = 2
	MESS_DETAIL = 3
	MESS_FULL   = 4
	
	//виды сообщений
	TMESS_DEAUTH = 0				//деаутентификация()
	TMESS_VERSION = 1				//запрос версии
	TMESS_AUTH = 2					//аутентификация(генерация pid)
	TMESS_LOGIN = 3					//вход в профиль
	TMESS_NOTIFICATION = 4			//сообщение клиент
	TMESS_REQUEST = 5				//запрос на подключение
	TMESS_CONNECT = 6				//запрашиваем подключение у клиента
	TMESS_DISCONNECT = 7			//сообщаем об отключении клиенту
	TMESS_REG = 8					//регистрация профиля
	TMESS_CONTACT = 9 				//создание, редактирование, удаление
	TMESS_CONTACTS = 10				//запрос списка контактов
	TMESS_LOGOUT = 11				//выход из профиля
	TMESS_CONNECT_CONTACT = 12		//запрос подключения к конакту из профиля
	TMESS_STATUSES = 13				//запрос всех статусов
	TMESS_STATUS = 14				//запрос статуса
	TMESS_INFO_CONTACT = 15			//запрос информации о клиенте
	TMESS_INFO_ANSWER = 16			//ответ на запрос информации
	TMESS_MANAGE = 17				//запрос на управление(перезагрузка, обновление, переустановка)
	TMESS_PING = 18					//проверка состояния подключения
	TMESS_CONTACT_REVERSE = 19		//добавление себя в чужой профиль

	REGULAR = 0
	MASTER  = 1
	NODE    = 2

)



var(

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
	}

	//считаем всякую бесполезную информацию или нет
	counterData struct{
		currentPos time.Time

		CounterBytes       [24]uint64
		CounterConnections [24]uint64

		CounterDayWeekBytes       [7]uint64
		CounterDayWeekConnections [7]uint64

		CounterDayBytes       [31]uint64
		CounterDayConnections [31]uint64

		CounterDayYearBytes       [365]uint64
		CounterDayYearConnections [365]uint64

		CounterMonthBytes       [12]uint64
		CounterMonthConnections [12]uint64

		mutex sync.Mutex
	}

	//меню веб интерфейса админки
	menuAdmin = []itemMenu{
		{"Логи", "/admin/logs"},
		{"Настройки", "/admin/options"},
		{"Ресурсы", "/admin/resources"},
		{"Статистика", "/admin/statistics"},
		{"reVisit", "/resource/reVisit.exe"} }

	//меню веб интерфейса профиля
	menuProfile = []itemMenu{
		{"Профиль", "/profile/my"},
		{"reVisit", "/resource/reVisit.exe"} }

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
	neighbours	map[string]*Neighbour
	neighboursM sync.Mutex

	//текстовая расшифровка сообщений для логов
	messLogText = []string{
		"BLANK",
		"ERROR",
		"INFO",
		"DETAIL",
		"FULL" }

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
		{TMESS_CONTACT_REVERSE, processContactReverse} }

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
		{"options_get", processApiOptionsGet} }

	//список доступных vnc клиентов и выбранный по-умолчанию
	defaultVnc = 0
	arrayVnc  []VNC
)

//double pointer
type dConn struct {
	pointer [2]*net.Conn
}

type Neighbour struct {
	Id          string
	Name		string
	Ip          string
	LastVisible time.Time
}

type ProcessingWeb struct {
	Make string
	Processing func(w http.ResponseWriter, r *http.Request)
}

type ProcessingMessage struct {
	TMessage int
	Processing func(message Message, conn *net.Conn, curClient *Client, id string)
}

type Message struct {
	TMessage int
	Messages []string
}

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
	SizeBuff 	int

	//учетка для админ панели
	AdminLogin	string
	AdminPass	string

	//режим работы экземпляра сервера
	Mode		int

	//очевидно что флаг для отладки
	FDebug		bool
}

type VNC struct {
	FileServer string
	FileClient string

	//это команды используем для старта под админскими правами(обычно это создание сервиса)
	CmdStartServer string
	CmdStopServer string
	CmdInstallServer string
	CmdRemoveServer string
	CmdConfigServer string
	CmdManageServer string

	//это комнады используем для старта без админских прав
	CmdStartServerUser string
	CmdStopServerUser string
	CmdInstallServerUser string
	CmdRemoveServerUser string
	CmdConfigServerUser string
	CmdManageServerUser string

	//комнды для vnc клиента
	CmdStartClient string
	CmdStopClient string
	CmdInstallClient string
	CmdRemoveClient string
	CmdConfigClient string
	CmdManageClient string

	PortServerVNC string
	Link string
	Name string
	Version string
	Description string
}

type itemMenu struct {
	Capt string
	Link string
}

type Client struct {
	Serial	string
	Pid		string
	Pass	string
	Version string
	Salt	string //for password
	Profile *Profile

	Conn	*net.Conn
	Code 	string //for connection

	profiles sync.Map //профили которые содержат этого клиента в контактах(используем для отправки им информации о своем статусе)
}

type Profile struct {
	Email	string
	Pass	string

	Contacts *Contact

	clients	sync.Map //клиенты которые авторизовались в этот профиль(используем для отправки им информации о статусе или изменений контактов)

	//всякая информация
	Capt	string
	Tel		string
	Logo	string
}

type Contact struct {
	Id      int
	Caption string
	Type	string	//cont - контакт, fold - папка
	Pid     string
	Digest  string //но тут digest
	Salt	string

	Inner   *Contact
	Next    *Contact
}