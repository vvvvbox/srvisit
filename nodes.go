package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

func masterServer() {
	logAdd(MESS_INFO, "masterServer запустился")

	ln, err := net.Listen("tcp", ":"+options.MasterPort)
	if err != nil {
		logAdd(MESS_ERROR, "masterServer не смог занять порт")
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			logAdd(MESS_ERROR, "masterServer не смог занять сокет")
			break
		}

		go ping(&conn)
		go masterHandler(&conn)
	}

	ln.Close()
	logAdd(MESS_INFO, "masterServer остановился")
}

func masterHandler(conn *net.Conn) {
	id := randomString(MAX_LEN_ID_LOG)
	logAdd(MESS_INFO, id+" masterServer получил соединение")

	var curNode Node

	reader := bufio.NewReader(*conn)

	for {
		buff, err := reader.ReadBytes('}')

		if err != nil {
			logAdd(MESS_ERROR, id+" ошибка чтения буфера")
			break
		}

		logAdd(MESS_DETAIL, id+fmt.Sprint(" buff ("+strconv.Itoa(len(buff))+"): "+string(buff)))

		//удаляем мусор
		if buff[0] != '{' {
			logAdd(MESS_INFO, id+" masterServer удаляем мусор")
			if bytes.Index(buff, []byte("{")) >= 0 {
				buff = buff[bytes.Index(buff, []byte("{")):]
			} else {
				continue
			}
		}

		var message Message
		err = json.Unmarshal(buff, &message)
		if err != nil {
			logAdd(MESS_ERROR, id+" ошибка разбора json")
			time.Sleep(time.Millisecond * WAIT_IDLE)
			continue
		}

		logAdd(MESS_DETAIL, id+" "+fmt.Sprint(message))

		//обрабатываем полученное сообщение
		if len(processingAgent) > message.TMessage {
			if processingAgent[message.TMessage].Processing != nil {
				go processingAgent[message.TMessage].Processing(message, conn, &curNode, id) //от одного агента может много приходить сообщений, не тормозим их
			} else {
				logAdd(MESS_INFO, id+" нет обработчика для сообщения")
				time.Sleep(time.Millisecond * WAIT_IDLE)
			}
		} else {
			logAdd(MESS_INFO, id+" неизвестное сообщение")
			time.Sleep(time.Millisecond * WAIT_IDLE)
		}

	}
	(*conn).Close()

	//если есть id значит скорее всего есть в карте
	if len(curNode.Id) == 0 {
		nodes.Delete(curNode.Id)
	}

	logAdd(MESS_INFO, id+" masterServer потерял соединение с агентом")
}

func nodeClient() {

	logAdd(MESS_INFO, "nodeClient запустился")

	for {
		conn, err := net.Dial("tcp", options.MasterServer+":"+options.MasterPort)
		if err != nil {
			logAdd(MESS_ERROR, "nodeClient не смог подключиться: "+fmt.Sprint(err))
			time.Sleep(time.Second * WAIT_IDLE_AGENT)
			continue
		}

		master = &conn

		hostname, err := os.Hostname()
		if err != nil {
			hostname = randomString(MAX_LEN_ID_NODE)
		}
		sendMessage(&conn, TMESS_AGENT_AUTH, hostname, options.MasterPassword)

		go ping(&conn)

		reader := bufio.NewReader(conn)
		for {
			buff, err := reader.ReadBytes('}')

			if err != nil {
				logAdd(MESS_ERROR, "nodeClient ошибка чтения буфера: "+fmt.Sprint(err))
				break
			}

			logAdd(MESS_DETAIL, fmt.Sprint("buff ("+strconv.Itoa(len(buff))+"): "+string(buff)))

			//удаляем мусор
			if buff[0] != '{' {
				logAdd(MESS_INFO, "nodeClient удаляем мусор")
				if bytes.Index(buff, []byte("{")) >= 0 {
					logAdd(MESS_DETAIL, fmt.Sprint("buff ("+strconv.Itoa(len(buff))+"): "+string(buff)))
					buff = buff[bytes.Index(buff, []byte("{")):]
				} else {
					continue
				}
			}

			var message Message
			err = json.Unmarshal(buff, &message)
			if err != nil {
				logAdd(MESS_ERROR, "nodeClient ошибка разбора json: "+fmt.Sprint(err))
				time.Sleep(time.Millisecond * WAIT_IDLE)
				continue
			}

			logAdd(MESS_DETAIL, fmt.Sprint(message))

			//обрабатываем полученное сообщение
			if len(processingAgent) > message.TMessage {
				if processingAgent[message.TMessage].Processing != nil {
					go processingAgent[message.TMessage].Processing(message, &conn, nil, randomString(MAX_LEN_ID_LOG))
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
	//logAdd(MESS_INFO, "nodeClient остановился") //недостижимо???
}

func processAgentAuth(message Message, conn *net.Conn, curNode *Node, id string) {
	logAdd(MESS_INFO, id+" пришла авторизация агента")

	if options.Mode == REGULAR {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		(*conn).Close()
		return
	}

	if options.Mode == NODE {
		logAdd(MESS_ERROR, id+" пришел отзыв на авторизацию")
		return
	}

	time.Sleep(time.Millisecond * WAIT_IDLE)

	if len(message.Messages) != 2 {
		logAdd(MESS_ERROR, id+" не правильное кол-во полей")
		(*conn).Close()
		return
	}

	if message.Messages[1] != options.MasterPassword {
		logAdd(MESS_ERROR, id+" не правильный пароль")
		(*conn).Close()
		return
	}

	curNode.Conn = conn
	curNode.Name = message.Messages[0]
	curNode.Id = randomString(MAX_LEN_ID_NODE)
	curNode.Ip = (*conn).RemoteAddr().String()

	if sendMessage(conn, TMESS_AGENT_AUTH, curNode.Id) {
		nodes.Store(curNode.Id, curNode)
		logAdd(MESS_INFO, id+" авторизация агента успешна")
	}
}

func processAgentAnswer(message Message, conn *net.Conn, curNode *Node, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id+" пришел ответ на авторизацию агента")

	//todo добавить обработку
}

func processAgentAddCode(message Message, conn *net.Conn, curNode *Node, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id+" пришла информация о создании сессии")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id+" не правильное кол-во полей")
		return
	}

	connectPeers(message.Messages[0])
}

func processAgentDelCode(message Message, conn *net.Conn, curNode *Node, id string) {
	if options.Mode != NODE {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id+" пришла информация об удалении сессии")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id+" не правильное кол-во полей")
		return
	}

	disconnectPeers(message.Messages[0])
}

func processAgentNewConnect(message Message, conn *net.Conn, curNode *Node, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id+" пришла информация о новом соединения")

	//if len(message.Messages) != 1 {
	//	logAdd(MESS_ERROR, id + " не правильное кол-во полей")
	//	return
	//}
	//
	//code := message.Messages[0]
	//
	//value, exist := channels.Load(code)
	//if exist == false {
	//	logAdd(MESS_ERROR, id + " не ждем такого соединения " + code)
	//	disconnectPeers(code)
	//	return
	//}
	//peers := value.(*dConn)
	//
	//peers.mutex.Lock()
	//if peers.node[0] == nil {
	//	peers.node[0] = curNode
	//} else if peers.pointer[1] == nil {
	//	peers.node[1] = curNode
	//}
	//peers.mutex.Unlock()
	//
	////мы должны дождаться два соединения
	//var cWait = 0
	//for (peers.node[0] == nil || peers.node[1] == nil) && cWait < WAIT_COUNT {
	//	logAdd(MESS_INFO, id + " ожидаем пира для " + code)
	//	time.Sleep(time.Millisecond * WAIT_IDLE)
	//	cWait++
	//}
	//
	////если не дождались одного из пира
	//for peers.node[0] == nil || peers.node[1] == nil {
	//	logAdd(MESS_ERROR, id + " не дождались пира для " + code)
	//	disconnectPeers(code)
	//	return
	//}
	//
	////если они у одного агента, то ничего
	//if peers.node[0].Id == peers.node[1].Id {
	//	logAdd(MESS_INFO, id + " пиры у одного агента " + code)
	//	return
	//}
	//
	//logAdd(MESS_INFO, id + " отправили запрос на соединение агента к агенту " + code)
}

func processAgentDelConnect(message Message, conn *net.Conn, curNode *Node, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id+" пришла информация об удалении соединения")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id+" не правильное кол-во полей")
		return
	}

	disconnectPeers(message.Messages[0])
}

func processAgentAddBytes(message Message, conn *net.Conn, curNode *Node, id string) {
	if options.Mode != MASTER {
		logAdd(MESS_ERROR, id+" режим не поддерживающий агентов")
		return
	}

	logAdd(MESS_INFO, id+" пришла информация статистики")

	if len(message.Messages) != 1 {
		logAdd(MESS_ERROR, id+" не правильное кол-во полей")
		return
	}

	bytes, err := strconv.Atoi(message.Messages[0])
	if err == nil {
		addCounter(uint64(bytes))
	}
}

func sendMessageToNodes(TMessage int, Messages ...string) {
	nodes.Range(func(key interface{}, value interface{}) bool {
		node := value.(*Node)
		return sendMessage(node.Conn, TMessage, Messages...)
	})
}

func sendMessageToMaster(TMessage int, Messages ...string) {
	sendMessage(master, TMessage, Messages...)
}
