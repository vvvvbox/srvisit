package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"os"
	"bytes"
	"time"
)



func httpServer(){

	http.Handle("/", http.RedirectHandler("/welcome", 301))

	http.HandleFunc("/welcome", handleWelcome)
	http.HandleFunc("/resource/", handleResource)
	http.HandleFunc("/resources", handleResources)
	http.HandleFunc("/statistics", handleStatistics)
	http.HandleFunc("/logs", handleLogs)

	http.HandleFunc("/api", handleAPI)

	err := http.ListenAndServe(":" + options.HttpserverPort, nil)
	if err != nil {
		logAdd(MESS_ERROR, "webServer не смог занять порт: " + fmt.Sprint(err))
	}
}

func handleDefault(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("How are you?"))
}

func handleWelcome(w http.ResponseWriter, r *http.Request) {

	file, _ := os.Open("resource/revisit.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenu())
		w.Write(body)
		return
	}

}

func handleResources(w http.ResponseWriter, r *http.Request) {

	connectionsString := "<pre>"

	var buf1 string
	connectionsString = connectionsString + fmt.Sprintln("клиенты:")
	clients.Range(func (key interface {}, value interface {}) bool {

		if value.(*Client).Profile == nil {
			buf1 = "no auth"
		} else {
			buf1 = getInvisibleEmail(value.(*Client).Profile.Email)
		}

		connectionsString = connectionsString + fmt.Sprintln(key.(string), value.(*Client).Serial, value.(*Client).Version, (*value.(*Client).Conn).RemoteAddr(), buf1)

		value.(*Client).profiles.Range(func (key interface {}, value interface {}) bool {
			connectionsString = connectionsString + fmt.Sprintln("\t ->", getInvisibleEmail(key.(string)))
			return true
		})

		return true
	})

	connectionsString = connectionsString + fmt.Sprintln("\n\nсессии:")
	channels.Range(func (key interface {}, value interface {}) bool {
		connectionsString = connectionsString + fmt.Sprintln((*value.(*dConn).pointer[0]).RemoteAddr(), "<->", (*value.(*dConn).pointer[1]).LocalAddr(), "<->", (*value.(*dConn).pointer[1]).RemoteAddr() )
		return true
	})

	connectionsString = connectionsString + fmt.Sprintln("\n\nпрофили:")
	profiles.Range(func (key interface {}, value interface {}) bool {

		connectionsString = connectionsString + fmt.Sprintln(getInvisibleEmail(key.(string)) )//(*value.(*Profile)).Pass)

		value.(*Profile).clients.Range(func (key interface {}, value interface {}) bool {
			connectionsString = connectionsString + fmt.Sprintln("\t", "<- " + key.(string) )
			return true
		})

		return true
	})
	connectionsString = connectionsString + "</pre>"

	file, _ := os.Open("resource/resources.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenu())
		body = pageReplace(body, "$connections", connectionsString)
		w.Write(body)
		return
	}

}

func handleStatistics(w http.ResponseWriter, r *http.Request) {

	file, _ := os.Open("resource/statistics.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenu())
		charts :=  getCounterBytes()
		body = pageReplace(body, "$headers", charts[0])
		body = pageReplace(body, "$values1", charts[1])
		body = pageReplace(body, "$values2", charts[2])

		w.Write(body)
		return
	}

}

func handleLogs(w http.ResponseWriter, r *http.Request) {

	file, _ := os.Open("resource/logs.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenu())
		//body = pageReplace(body, "$logs", logsString)
		w.Write(body)
		return
	}

}

func handleAPI(w http.ResponseWriter, r *http.Request) {

	make, exist := r.URL.Query()["make"]
	if exist == true {
		if len(make) > 0 && make[0] == "defaultvnc" {
			logAdd(MESS_INFO, "WEB Запрос vnc версии по-умолчанию")
			if default_vnc != -1 {
				buff, err := json.Marshal(array_vnc[default_vnc])
				if err != nil {
					logAdd(MESS_ERROR, "WEB Не получилось отправить версию VNC")
					return
				}
				w.Write(buff)
				return
			}
		} else if len(make) > 0 && make[0] == "listvnc" {
			logAdd(MESS_INFO, "WEB Запрос списка vnc")
			buff, err := json.Marshal(array_vnc)
			if err != nil {
				logAdd(MESS_ERROR, "WEB Не получилось отправить список VNC")
				return
			}
			w.Write(buff)
			return
		} else if len(make) > 0 && make[0] == "getlog" {
			logAdd(MESS_INFO, "WEB Запрос log")
			file, _ := os.Open(LOG_NAME)
			log, err := ioutil.ReadAll(file)
			if err == nil {
				file.Close()
			}
			w.Write(log)
			return
		} else if len(make) > 0 && make[0] == "clearlog" {
			logAdd(MESS_INFO, "WEB Запрос очистки log")
			if logFile != nil {
				logFile.Close()
				logFile = nil
			}
			http.Redirect(w, r, "/logs", http.StatusTemporaryRedirect)
			return
		} else {
			logAdd(MESS_ERROR, "WEB Нет такого действия")
		}
	}

	logAdd(MESS_ERROR, "WEB Что-то пошло не так")
	w.WriteHeader(http.StatusBadRequest)
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}



func getCounterBytes() []string {

	h := time.Now().Hour() + 1

	values1 := append(counterData.counterBytes[h:], counterData.counterBytes[:h]...)
	values2 := append(counterData.counterConnect[h:], counterData.counterConnect[:h]...)

	for i := 0; i < 24; i++ {
		values1[i] = values1[i] / 2
		values2[i] = values2[i] / 2
	}

	headers := make([]int, 0)
	for i := h; i < 24; i++ {
		headers = append(headers, i)
	}
	for i := 0; i < h; i++ {
		headers = append(headers, i)
	}

	stringHeaders := "["
	for i := 0; i < 24; i++ {
		stringHeaders = stringHeaders + "'" + fmt.Sprint(headers[i]) + "'"
		if i != 23 {
			stringHeaders = stringHeaders  + ", "
		}
	}
	stringHeaders = stringHeaders + "]"

	stringValues1 := "["
	for i := 0; i < 24; i++ {
		stringValues1 = stringValues1 + fmt.Sprint(values1[i] / 1024 ) //in Kb
		if i != 23 {
			stringValues1 = stringValues1 + ", "
		}
	}
	stringValues1 = stringValues1 + "]"

	stringValues2 := "["
	for i := 0; i < 24; i++ {
		stringValues2 = stringValues2 + fmt.Sprint(values2[i])
		if i != 23 {
			stringValues2 = stringValues2 + ", "
		}
	}
	stringValues2 = stringValues2 + "]"

	answer := make([]string, 0)
	answer = append(answer, stringHeaders)
	answer = append(answer, stringValues1)
	answer = append(answer, stringValues2)
	return answer
}

func pageReplace(e []byte, a string, b string) []byte{
	return bytes.Replace(e, []byte(a), []byte(b), -1)
}

func addMenu() string{
	out, err := json.Marshal(menus)
	if err == nil {
		return string(out)
	}

	return ""
}