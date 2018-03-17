package main

import (
	"net"
	"time"
	"encoding/json"
	"fmt"
	"bufio"
	"strconv"
	"bytes"
	"os"
)



func nodeClient(){

	logAdd(MESS_INFO, "nodeClient запустился")

	for {
		conn, err := net.Dial("tcp", options.MasterServer+":"+options.MainServerPort)
		if err != nil {
			logAdd(MESS_ERROR, "nodeClient не смог подключиться: " + fmt.Sprint(err))
			time.Sleep(time.Second * WAIT_IDLE_AGENT)
			continue
		}

		master = &conn

		hostname, err := os.Hostname()
		if err != nil {
			hostname = randomString(MAX_LEN_ID_NODE)
		}
		sendMessage(&conn, TMESS_AGENT_AUTH, hostname)

		go ping(&conn)


		reader := bufio.NewReader(conn)
		for {
			buff, err := reader.ReadBytes('}')

			if err != nil {
				logAdd(MESS_ERROR, "nodeClient ошибка чтения буфера: " + fmt.Sprint(err))
				break
			}

			logAdd(MESS_DETAIL, fmt.Sprint("buff (" + strconv.Itoa(len(buff)) + "): " + string(buff)))

			//удаляем мусор
			if buff[0] != '{' {
				logAdd(MESS_INFO, "nodeClient удаляем мусор")
				if bytes.Index(buff, []byte("{")) >= 0 {
					logAdd(MESS_DETAIL, fmt.Sprint("buff (" + strconv.Itoa(len(buff)) + "): " + string(buff)))
					buff = buff[bytes.Index(buff, []byte("{")):]
				} else {
					continue
				}
			}

			var message Message
			err = json.Unmarshal(buff, &message)
			if err != nil {
				logAdd(MESS_ERROR, "nodeClient ошибка разбора json: " + fmt.Sprint(err))
				time.Sleep(time.Millisecond * WAIT_IDLE)
				continue
			}

			logAdd(MESS_DETAIL, fmt.Sprint(message))

			//обрабатываем полученное сообщение
			if len(processing) > message.TMessage {
				if processing[message.TMessage].Processing != nil {
					processing[message.TMessage].Processing(message, &conn, nil, randomString(MAX_LEN_ID_LOG))
				} else {
					logAdd(MESS_INFO, "nodeClient нет обработчика для сообщения")
					time.Sleep(time.Millisecond * WAIT_IDLE)
				}
			} else {
				logAdd(MESS_INFO, "nodeClient неизвестное сообщение")
				time.Sleep(time.Millisecond * WAIT_IDLE)
			}

		}
		conn.Close()
	}
	logAdd(MESS_INFO, "nodeClient остановился")
}

func processAgentAuth(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла авторизация агента")

	time.Sleep(time.Millisecond * WAIT_IDLE)

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	var node Node
	node.Conn = conn
	node.Name = message.Messages[0]
	node.Id = randomString(MAX_LEN_ID_NODE)
	node.Ip = (*conn).RemoteAddr().String()

	if sendMessage(conn, TMESS_AGENT_AUTH, node.Id){
		nodes.Store(node.Id, &node)
		logAdd(MESS_INFO, id + " авторизация агента успешна")
	}
}

func processAgentAnswer(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла ответ на авторизацию агента")

	//todo добавить обработку
}

func processAgentAddCode(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла информация о создании сессии")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	connectPeers(message.Messages[0])
}

func processAgentDelCode(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла информация об удалении сессии")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	disconnectPeers(message.Messages[0])
}

func processAgentNewConnect(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла информация об новом соединения")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	//мы должны дождаться два соединения
	//если они у одного агента, то ничего
	//если они у разных агентов, то одному надо отправить команду подключиться к другому

	value, exist := channels.Load(message.Messages[0])
	if exist {
		peers := value.(*dConn)
		peers.mutex.Lock()
		//var numPeer int
		//if peers.node[0] == nil {
		//	peers.node[0] = conn
		//	numPeer = 1
		//} else if peers.pointer[1] == nil {
		//	peers.node[1] = conn
		//	numPeer = 0
		//}
		peers.mutex.Unlock()

	}

	disconnectPeers(message.Messages[0])
}

func processAgentDelConnect(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла информация об удалении соединения")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	disconnectPeers(message.Messages[0])
}

func processAgentAddBytes(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла информация статистики")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	bytes, err := strconv.Atoi(message.Messages[0])
	if err == nil {
		addCounter(uint64(bytes))
	}
}

func processAgentConnect(message Message, conn *net.Conn, curClient *Client, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id + " режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id + " пришла информация статистики")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id + " не правильное кол-во полей")
		return
	}

	//todo подключаемся к другому агента
	//111111111111111111111111111111111111
	//111111111111111111111111111111111111
	//111111111111111111111111111111111111
	//111111111111111111111111111111111111
}



func sendMessageToNodes(TMessage int, Messages... string) {
	nodes.Range(func(key interface{}, value interface{}) bool{
		node := value.(*Node);
		return sendMessage(node.Conn, TMessage, Messages...);
	})
}

func sendMessageToMaster(TMessage int, Messages... string) {
	sendMessage(master, TMessage, Messages...);
}