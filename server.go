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
	"io"
	"strings"
)



func mainServer(){
	logAdd(MESS_INFO, "mainServer запустился")

	ln, err := net.Listen("tcp", ":" + options.MainserverPort)
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

		defer conn.Close()

		go pinger(&conn)
		go mainHandler(&conn)
	}

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
		curClient.Profile.clients.Delete(strings.Replace(curClient.Pid, ":", "", -1))
	}

	//пробежимся по профилям где мы есть и отправим новый статус
	curClient.profiles.Range(func (key interface {}, value interface {}) bool {
		profile := *value.(*Profile)

		//все кто авторизовался в этот профиль должен получить новый статус
		profile.clients.Range(func (key interface {}, value interface{}) bool {
			client := value.(*Client)
			sendMessage(client.Conn, TMESS_STATUS, curClient.Pid, "0")
			return true
		})

		return true
	})

	logAdd(MESS_INFO, id + " mainServer потерял соединение")
	if curClient.Pid != "" {
		clients.Delete(strings.Replace(curClient.Pid, ":", "", -1))
	}
}

func dataServer(){
	logAdd(MESS_INFO, "dataServer запустился")

	ln, err := net.Listen("tcp", ":" + options.DataserverPort)
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

		defer conn.Close()
		go dataHandler(&conn)
	}

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
		var npeer int
		if peers.pointer[0] == nil {
			peers.pointer[0] = conn
			npeer = 1
		} else if peers.pointer[1] == nil {
			peers.pointer[1] = conn
			npeer = 0
		}

		var cwait = 0
		for peers.pointer[npeer] == nil && cwait < WAIT_COUNT{
			logAdd(MESS_INFO, id + " ожидаем пира для " + code)
			time.Sleep(time.Millisecond * WAIT_IDLE)
			cwait++
		}

		if peers.pointer[npeer] == nil {
			logAdd(MESS_ERROR, id + " превышено время ожидания")
			channels.Delete(code)
			break
		}

		logAdd(MESS_INFO, id + " пир существует для " + code)
		time.Sleep(time.Millisecond * WAIT_AFTER_CONNECT)

		var z []byte
		z = make([]byte, options.SizeBuff)

		var bytes uint64

		for {
			n1, err1 := (*conn).Read(z)
			if err1 != nil {
				logAdd(MESS_ERROR, id + " " + fmt.Sprint(err1))
			}
			bytes = bytes + uint64(n1)

			n2, err2 := (*peers.pointer[npeer]).Write(z[:n1])
			if err2 != nil {
				logAdd(MESS_ERROR, id + " "  + fmt.Sprint(err2))
			}
			bytes = bytes + uint64(n2)

			if (err1 != nil && err1 != io.EOF) && err2 != nil || n1 == 0 || n2 == 0 {
				logAdd(MESS_INFO, id + " соединение закрылось: " + fmt.Sprint(n1, n2))
				(*peers.pointer[npeer]).Close()
				break
			}
		}

		addCounter(bytes)

		logAdd(MESS_INFO, id + " поток завершается")
		channels.Delete(code)
		break

	}
	(*conn).Close()
	logAdd(MESS_INFO, id + " dataHandler потерял соединение")

}



func pinger(conn *net.Conn){
	success := true
	for success{
		time.Sleep(time.Second * WAIT_PING)
		success = sendMessage(conn, TMESS_PING)
	}
}