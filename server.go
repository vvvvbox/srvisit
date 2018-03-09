package main

import (
	"net"
	"os"
	"bufio"
	"fmt"
	"time"
	"strconv"
	"bytes"
	"encoding/json"
	"strings"
)



func mainServer(){
	logAdd(MESS_INFO, "mainServer запустился")

	ln, err := net.Listen("tcp", ":" + options.MainServerPort)
	if err != nil {
		logAdd(MESS_ERROR, "mainServer не смог занять порт")
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			logAdd(MESS_ERROR, "mainServer не смог занять сокет")
			break
		}

		go ping(&conn)
		go mainHandler(&conn)
	}

	ln.Close()
	logAdd(MESS_INFO, "mainServer остановился")
}

func mainHandler(conn *net.Conn) {
	id := randomString(6)
	logAdd(MESS_INFO, id + " mainServer получил соединение")

	var curClient Client

	reader := bufio.NewReader(*conn)

	for {
		buff, err := reader.ReadBytes('}')

		if err != nil {
			logAdd(MESS_ERROR, id + " ошибка чтения буфера")
			break
		}

		logAdd(MESS_DETAIL, id + fmt.Sprint(" buff (" + strconv.Itoa(len(buff)) + "): " + string(buff)))

		//удаляем мусор
		if buff[0] != '{' {
			logAdd(MESS_INFO, id + " mainServer удаляем мусор")
			if bytes.Index(buff, []byte("{")) >= 0 {
				buff = buff[bytes.Index(buff, []byte("{")):]
			} else {
				continue
			}
		}

		var message Message
		err = json.Unmarshal(buff, &message)
		if err != nil {
			logAdd(MESS_ERROR, id + " ошибка разбора json")
			time.Sleep(time.Millisecond * WAIT_IDLE)
			continue
		}

		logAdd(MESS_DETAIL, id + " " + fmt.Sprint(message))

		//обрабатываем полученное сообщение
		if len(processing) > message.TMessage{
			if processing[message.TMessage].Processing != nil{
				processing[message.TMessage].Processing(message, conn, &curClient, id)
			} else {
				logAdd(MESS_INFO, id + " нет обработчика для сообщения")
				time.Sleep(time.Millisecond * WAIT_IDLE)
			}
		} else {
			logAdd(MESS_INFO, id + " неизвестное сообщение")
			time.Sleep(time.Millisecond * WAIT_IDLE)
		}

	}
	(*conn).Close()

	//удалим себя из профиля если авторизованы
	if curClient.Profile != nil {
		curClient.Profile.clients.Delete(cleanPid(curClient.Pid))
	}

	//пробежимся по профилям где мы есть и отправим новый статус
	curClient.profiles.Range(func (key interface {}, value interface {}) bool {
		profile := *value.(*Profile)

		//все кто авторизовался в этот профиль должен получить новый статус
		profile.clients.Range(func (key interface {}, value interface{}) bool {
			client := value.(*Client)
			sendMessage(client.Conn, TMESS_STATUS, cleanPid(curClient.Pid), "0")
			return true
		})

		return true
	})

	logAdd(MESS_INFO, id + " mainServer потерял соединение")
	if curClient.Pid != "" {
		clients.Delete(cleanPid(curClient.Pid))
	}
}

func dataServer(){
	logAdd(MESS_INFO, "dataServer запустился")

	ln, err := net.Listen("tcp", ":" + options.DataServerPort)
	if err != nil {
		logAdd(MESS_ERROR, "dataServer не смог занять порт")
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			logAdd(MESS_ERROR, "dataServer не смог занять сокет")
			break
		}

		go dataHandler(&conn)
	}

	ln.Close()
	logAdd(MESS_INFO, "dataServer остановился")
}

func dataHandler(conn *net.Conn) {
	id := randomString(6)
	logAdd(MESS_INFO, id + " dataHandler получил соединение")

	for {
		code, err := bufio.NewReader(*conn).ReadString('\n')

		if err != nil {
			logAdd(MESS_ERROR, id + " ошибка чтения кода")
			break
		}

		code = code[:len(code) - 1]
		value, exist := channels.Load(code)
		if exist == false {
			logAdd(MESS_ERROR, id + " не ожидаем такого кода")
			break
		}

		peers := value.(*dConn)
		var numPeer int
		if peers.pointer[0] == nil {
			peers.pointer[0] = conn
			numPeer = 1
		} else if peers.pointer[1] == nil {
			peers.pointer[1] = conn
			numPeer = 0
		}

		var cWait = 0
		for peers.pointer[numPeer] == nil && cWait < WAIT_COUNT{
			logAdd(MESS_INFO, id + " ожидаем пира для " + code)
			time.Sleep(time.Millisecond * WAIT_IDLE)
			cWait++
		}

		if peers.pointer[numPeer] == nil {
			logAdd(MESS_ERROR, id + " превышено время ожидания")
			channels.Delete(code)
			break
		}

		logAdd(MESS_INFO, id + " пир существует для " + code)
		time.Sleep(time.Millisecond * WAIT_AFTER_CONNECT)

		var z []byte
		z = make([]byte, options.SizeBuff)

		var countBytes uint64
		var n1, n2 int
		var err1, err2 error

		for {
			n1, err1 = (*conn).Read(z)

			if peers.pointer[numPeer] == nil {
				logAdd(MESS_INFO, id + " потеряли пир")
				time.Sleep(time.Millisecond * WAIT_AFTER_CONNECT)
				break
			}

			n2, err2 = (*peers.pointer[numPeer]).Write(z[:n1])

			countBytes = countBytes + uint64(n1 + n2)

			if err1 != nil || err2 != nil || n1 == 0 || n2 == 0 {
				logAdd(MESS_INFO, id + " соединение закрылось: " + fmt.Sprint(n1, n2))
				logAdd(MESS_INFO, id + " err1: "  + fmt.Sprint(err1))
				logAdd(MESS_INFO, id + " err2: "  + fmt.Sprint(err2))
				time.Sleep(time.Millisecond * WAIT_AFTER_CONNECT)
				if peers.pointer[numPeer] != nil {
					(*peers.pointer[numPeer]).Close()
				}
				break
			}
		}

		addCounter(countBytes)

		logAdd(MESS_INFO, id + " поток завершается")
		channels.Delete(code)
		break

	}
	(*conn).Close()
	logAdd(MESS_INFO, id + " dataHandler потерял соединение")

}

func disconnectPeers(code string) {
	value, exists := channels.Load(code)
	if exists {
		pair := value.(*dConn)
		channels.Delete(code)

		if pair.pointer[0] != nil {
			(*pair.pointer[0]).Close()
		}
		if pair.pointer[1] != nil {
			(*pair.pointer[1]).Close()
		}
	}
}



func finderNeighbours() {
	neighbours = make(map[string]*Neighbour)

	//там чистим не активные агенты
	go cleanerNeighbours()

	//здесь мы ждём подключения
	addr, err := net.ResolveUDPAddr("udp", ":" + fmt.Sprint(PORT_FINDER_NEIGHBOURS))
	checkError(err)

	conn, err := net.ListenUDP("udp", addr)
	checkError(err)

	go agentFinderNeighbours(conn)

	reader := bufio.NewReader(conn)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			logAdd(MESS_ERROR, "Ошибка чтения сообщения от агента: " + fmt.Sprint(err))
			fmt.Println(err)
			continue
		}

		n, ip, id := parseAnswerAgent(string(line))

		neighbour := neighbours[ip]
		if neighbour == nil {
			var newNeighbour Neighbour
			newNeighbour.Name = n
			newNeighbour.Id = id
			newNeighbour.Ip = ip
			newNeighbour.LastVisible = time.Now()
			neighboursM.Lock()
			neighbours[ip] = &newNeighbour
			neighboursM.Unlock()
			logAdd(MESS_INFO, "Появился новый агент " + ip + " " + id + " " + n)
		} else {
			neighbour.LastVisible = time.Now()
			//logAdd(MESS_DETAIL, "Обновили состояние агента " + ip)
		}
	}
}

func agentFinderNeighbours(conn *net.UDPConn) {

	hostname, _ := os.Hostname()
	id := randomString(MAX_LEN_ID_NEIGHBOUR)

	ip := getMyIp()

	myString := hostname + ":" + ip + ":" + fmt.Sprint(id) + "\n"

	//периодически делаем рассылку о своём существовании
	for {
		adr, err := net.ResolveUDPAddr("udp", fmt.Sprint(net.IPv4bcast) + ":" + fmt.Sprint(PORT_FINDER_NEIGHBOURS))
		checkError(err)

		n, err := (*conn).WriteToUDP([]byte(myString), adr)
		checkError(err)

		if n <= 0 {
			logAdd(MESS_ERROR, "Не получилось отправить агенту сообщение для соседей")
		}

		time.Sleep(time.Second * WAIT_IDLE_FINDER)
	}
}

func parseAnswerAgent(message string) (string, string, string){
	p1 := strings.Index(message, ":")
	p2 := strings.LastIndex(message, ":")

	if p1 == p2 {
		logAdd(MESS_ERROR, "Ошибка разбора сообщения от агента")
		return "", "", ""
	}

	return message[:p1], message[p1 + 1:p2], message[p2 + 1:]
}

func cleanerNeighbours() {
	for {
		for _, agent := range neighbours {
			if agent.LastVisible.Add(time.Second * WAIT_IDLE_CLEANER).Before(time.Now()) {
				neighboursM.Lock()
				delete(neighbours, agent.Ip)
				neighboursM.Unlock()
				logAdd(MESS_INFO, "Удалили устаревшего агента " + agent.Ip + " " + agent.Id + " " + agent.Name)
			}
		}

		time.Sleep(time.Second * WAIT_IDLE_CLEANER)
	}
}

func getMyIp() string {
	int, err := net.Interfaces()
	checkError(err)

	ip := net.IPv4zero.String()
	for _, i := range int {
		if (i.Flags&net.FlagLoopback == 0) && (i.Flags&net.FlagPointToPoint == 0) && (i.Flags&net.FlagUp == 1) {
			z, err := i.Addrs()
			checkError(err)

			for _, j := range z {
				x, _, _ := net.ParseCIDR(j.String())

				if x.IsGlobalUnicast() && x.To4() != nil {
					ip = x.To4().String()
					return ip
				}
			}
		}
	}

	return ip
}



func ping(conn *net.Conn){
	success := true
	for success{
		time.Sleep(time.Second * WAIT_PING)
		success = sendMessage(conn, TMESS_PING)
	}
}