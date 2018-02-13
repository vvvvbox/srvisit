package main

import (
	"net"
	"sync"
	"os"
)

const(
	REVISIT_VERSION = 0.1

	//общие константы
	CODE_LENGTH = 64 //длина code
	PASSWORD_LENGTH = 14
	FILE_PROFILES = "profiles.list"
	FILE_OPTIONS = "options.cfg"
	FILE_VNCLIST = "vnc.list"
	LOG_NAME = "log.txt"

	//константы ожидания
	WAIT_COUNT = 15
	WAIT_IDLE = 500
	WAIT_AFTER_CONNECT = 250
	WAIT_HELPER_CYCLE = 5
	WAIT_PING = 10

	//виды сообщений логов
	MESS_ERROR = 1
	MESS_INFO = 2
	MESS_DETAIL = 3
	MESS_FULL = 4

	//виды сообщений
	TMESS_DEAUTH = 0
	TMESS_VERSION = 1
	TMESS_AUTH = 2
	TMESS_LOGIN = 3
	TMESS_NOTIFICATION = 4
	TMESS_REQUEST = 5
	TMESS_CONNECT = 6
	TMESS_DISCONNECT = 7
	TMESS_REG = 8
	TMESS_CONTACT = 9 				//создание, редактирование, удаление
	TMESS_CONTACTS = 10
	TMESS_LOGOUT = 11
	TMESS_CONNECT_CONTACT = 12
	TMESS_STATUSES = 13
	TMESS_STATUS = 14
	TMESS_INFO_CONTACT = 15
	TMESS_INFO_ANSWER = 16
	TMESS_MANAGE = 17
	TMESS_PING = 18

)

var(

	//опции по-умолчанию
	options = Options{
		"",
		"",
		"",
		"",
		"65471",
		"65475",
		"8090",
		16000,
		true,
	}

	//считаем всякую бесполезную информацию или нет
	//fCounter = true
	counterData struct{
		currentPos int
		counterBytes [24]uint64
		counterConnect [24]uint64

		mutex sync.Mutex
	}

	//меню веб интерфейса
	menus = []itemMenu{
		{"Логи", "/logs"},
		{"Ресурсы", "/resources"},
		{"Статистика", "/statistics"},
		{"reVisit", "/"} }

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
		{TMESS_VERSION, nil},
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
		{TMESS_PING, processPing} }

	//список доступных vnc клиентов и выбранный по-умолчанию
	default_vnc = -1
	array_vnc []VNC
)

//double pointer
type dConn struct {
	pointer [2]*net.Conn
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
	MainserverPort string

	//реквизиты сервер
	DataserverPort string

	//реквизиты веб сервера
	HttpserverPort string

	//размер буфера для операций с сокетами
	SizeBuff 	int

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