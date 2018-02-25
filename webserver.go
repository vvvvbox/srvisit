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

	http.Handle("/admin", http.RedirectHandler("/admin/welcome", 301))
	http.HandleFunc("/admin/welcome", handleWelcome)
	http.HandleFunc("/admin/resources", handleResources)
	http.HandleFunc("/admin/statistics", handleStatistics)
	http.HandleFunc("/admin/logs", handleLogs)

	http.Handle("/", http.RedirectHandler("/profile/welcome", 301))
	http.Handle("/profile", http.RedirectHandler("/profile/welcome", 301))
	http.HandleFunc("/profile/welcome", handleProfileWelcome)
	http.HandleFunc("/profile/my", handleProfileMy)

	http.HandleFunc("/resource/", handleResource)
	http.HandleFunc("/api", handleAPI)

	err := http.ListenAndServe(":" + options.HttpServerPort, nil)
	if err != nil {
		logAdd(MESS_ERROR, "webServer не смог занять порт: " + fmt.Sprint(err))
	}
}



//хэндлеры для профиля
func handleProfileWelcome(w http.ResponseWriter, r *http.Request) {

	file, _ := os.Open("resource/profile/welcome.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenuProfile())
		w.Write(body)
		return
	}

}

func handleProfileMy(w http.ResponseWriter, r *http.Request) {
	curProfile := checkProfileAuth(w, r)

	if curProfile == nil {
		return
	}

	file, _ := os.Open("resource/profile/my.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenuProfile())
		w.Write(body)
		return
	}
}



//хэндлеры для админки
func handleWelcome(w http.ResponseWriter, r *http.Request) {

	file, _ := os.Open("resource/admin/welcome.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenuAdmin())
		w.Write(body)
		return
	}

}

func handleResources(w http.ResponseWriter, r *http.Request) {

	if !checkAdminAuth(w, r) {
		return
	}

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

	file, _ := os.Open("resource/admin/resources.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenuAdmin())
		body = pageReplace(body, "$connections", connectionsString)
		w.Write(body)
		return
	}

}

func handleStatistics(w http.ResponseWriter, r *http.Request) {

	file, _ := os.Open("resource/admin/statistics.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenuAdmin())
		charts :=  getCounterBytes()
		body = pageReplace(body, "$headers", charts[0])
		body = pageReplace(body, "$values1", charts[1])
		body = pageReplace(body, "$values2", charts[2])

		w.Write(body)
		return
	}

}

func handleLogs(w http.ResponseWriter, r *http.Request) {

	if !checkAdminAuth(w, r) {
		return
	}

	file, _ := os.Open("resource/admin/logs.html")
	body, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()

		body = pageReplace(body, "$menu", addMenuAdmin())
		//body = pageReplace(body, "$logs", logsString)
		w.Write(body)
		return
	}

}



//ресурсы и api
func handleResource(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}

func handleAPI(w http.ResponseWriter, r *http.Request) {

	actMake := r.URL.Query().Get("make")

	for _, m := range processingWeb {
		if actMake == m.Make {
			if m.Processing != nil {
				m.Processing(w, r)
			} else {
				logAdd(MESS_INFO, "WEB Нет обработчика для сообщения")
				time.Sleep(time.Millisecond * WAIT_IDLE)
			}
			return
		}
	}

	time.Sleep(time.Millisecond * WAIT_IDLE)
	logAdd(MESS_ERROR, "WEB Неизвестное сообщение")
	http.Error(w, "bad request", http.StatusBadRequest)
}



//раскрытие api
func processApiDefaultVnc(w http.ResponseWriter, r *http.Request) {
	logAdd(MESS_INFO, "WEB Запрос vnc версии по-умолчанию")

	if len(arrayVnc) < defaultVnc {
		buff, err := json.Marshal(arrayVnc[defaultVnc])
		if err != nil {
			logAdd(MESS_ERROR, "WEB Не получилось отправить версию VNC")
			return
		}
		w.Write(buff)
		return
	}
	http.Error(w, "vnc is not prepared", http.StatusNotAcceptable)
}

func processApiListVnc(w http.ResponseWriter, r *http.Request) {
	logAdd(MESS_INFO, "WEB Запрос списка vnc")

	buff, err := json.Marshal(arrayVnc)
	if err != nil {
		logAdd(MESS_ERROR, "WEB Не получилось отправить список VNC")
		return
	}
	w.Write(buff)
}

func processApiGetLog(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(w, r) {
		return
	}

	logAdd(MESS_INFO, "WEB Запрос log")
	file, _ := os.Open(LOG_NAME)
	log, err := ioutil.ReadAll(file)
	if err == nil {
		file.Close()
	}
	w.Write(log)
}

func processApiClearLog(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(w, r) {
		return
	}

	logAdd(MESS_INFO, "WEB Запрос очистки log")
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	http.Redirect(w, r, "/logs", http.StatusTemporaryRedirect)
}

func processApiProfileSave(w http.ResponseWriter, r *http.Request) {
	curProfile := checkProfileAuth(w, r)
	if curProfile == nil {
		return
	}

	logAdd(MESS_INFO, "WEB Запрос сохранения профиля " + curProfile.Email)

	pass1 := string(r.FormValue("abc"))
	pass2 := string(r.FormValue("def"))

	capt := string(r.FormValue("capt"))
	tel := string(r.FormValue("tel"))
	logo := string(r.FormValue("logo"))

	if (pass1 != "*****") && (len(pass1) > 3) && (pass1 == pass2){
		curProfile.Pass = pass1
	}
	curProfile.Capt = capt
	curProfile.Tel = tel
	curProfile.Logo = logo

	handleProfileMy(w, r)
}

func processApiProfileGet(w http.ResponseWriter, r *http.Request) {
	curProfile := checkProfileAuth(w, r)
	if curProfile == nil {
		return
	}

	logAdd(MESS_INFO, "WEB Запрос информации профиля " + curProfile.Email)

	newProfile := *curProfile
	newProfile.Pass = "*****"
	b, err := json.Marshal(newProfile)
	if err == nil {
		w.Write(b)
		return
	}

	http.Error(w, "", http.StatusBadRequest)
}



//общие функции
func checkProfileAuth(w http.ResponseWriter, r *http.Request) *Profile {

	user, pass, ok := r.BasicAuth()

	if ok {
		value, exist := profiles.Load(user)

		if exist {
			if value.(*Profile).Pass == pass {
				//logAdd(MESS_INFO, "Аутентификация успешна " + user + "/"+ r.RemoteAddr)
				return value.(*Profile)
			}
		}
	}

	logAdd(MESS_ERROR, "Аутентификация провалилась " + r.RemoteAddr)
	w.Header().Set("WWW-Authenticate", "Basic")
	http.Error(w, "auth req", http.StatusUnauthorized)
	return nil
}

func checkAdminAuth(w http.ResponseWriter, r *http.Request) bool {

	user, pass, ok := r.BasicAuth()
	if ok {
		if user == options.AdminLogin && pass == options.AdminPass {
			return true
		}
	}

	w.Header().Set("WWW-Authenticate", "Basic")
	http.Error(w, "auth req", http.StatusUnauthorized)
	return false
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

func addMenuAdmin() string{
	out, err := json.Marshal(menuAdmin)
	if err == nil {
		return string(out)
	}

	return ""
}

func addMenuProfile() string{
	out, err := json.Marshal(menuProfile)
	if err == nil {
		return string(out)
	}

	return ""
}