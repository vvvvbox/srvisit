package main

import (
	"fmt"
	"time"
	"strconv"
	"bytes"
	"math/rand"
	"encoding/json"
	"net"
	"crypto/sha256"
	"os"
	"io/ioutil"
	"strings"
)



func helperThread(){
	logAdd(MESS_INFO, "helperThread запустился")
	for true {
		saveProfiles()
		swiftCounter()

		time.Sleep(time.Second * WAIT_HELPER_CYCLE)
	}
	logAdd(MESS_INFO, "helperThread закончил работу")
}

func getPid(serial string) string{

	var a uint64 = 1
	for _, f := range serial {
		a = a * uint64(f)
	}

	//todo добавить нули если число меньше трех знаков
	b := a % 999
	for b < 100 {
		b = b * 10
	}
	c := (a / 999) % 999
	for c < 100 {
		c = c * 10
	}
	d := ((a / 999) / 999 ) % 999
	for d < 100 {
		d = d * 10
	}
	e := (((a / 999) / 999 ) / 999 ) % 999
	for e < 100 {
		e = e * 10
	}

	var r string
	r = strconv.Itoa(int(b)) + ":" + strconv.Itoa(int(c)) + ":" + strconv.Itoa(int(d)) + ":" + strconv.Itoa(int(e))

	return r
}

func logAdd(tmess int, mess string){
	if options.FDebug && typeLog >= tmess {

		if logFile == nil {
			logFile, _ = os.Create(LOG_NAME)
		}

		//todo наверное стоит убрать, но пока меашет пинг в логах
		if strings.Contains(mess, "buff (31): {\"TMessage\":18,\"Messages\":null}") || strings.Contains(mess, "{18 []}") {
			return
		}

		logFile.Write([]byte(fmt.Sprint(time.Now().Format("02 Jan 2006 15:04:05.000000")) + "\t" + messLogText[tmess] + ":\t" + mess + "\n"))

		fmt.Println(fmt.Sprint(time.Now().Format("02 Jan 2006 15:04:05.000000")) + "\t" + messLogText[tmess] + ":\t" + mess)
	}

}

func createMessage(TMessage int, Messages ...string) Message{
	var mes Message
	mes.TMessage = TMessage
	mes.Messages = Messages
	return mes
}

func randomString(l int) string {
	var result bytes.Buffer
	var temp string
	for i := 0; i < l; {
		if string(randInt(65, 90)) != temp {
			temp = string(randInt(65, 90))
			result.WriteString(temp)
			i++
		}
	}
	return result.String()
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}

func sendMessageRaw(conn *net.Conn, TMessage int, Messages[] string) bool{
	if conn == nil {
		logAdd(MESS_ERROR, "нет сокета для отправки")
		return false
	}

	var mes Message
	mes.TMessage = TMessage
	mes.Messages = Messages

	out, err := json.Marshal(mes)
	if err == nil && conn != nil {
		_, err = (*conn).Write(out)
		if err == nil {
			return true
		}
	}
	return false
}

func sendMessage(conn *net.Conn, TMessage int, Messages ...string) bool{
	return sendMessageRaw(conn, TMessage, Messages)
}

func getSHA256(str string) string {

	s := sha256.Sum256([]byte(str))
	var r string

	for _, x := range s {
		r = r + fmt.Sprintf("%02x", x)
	}

	return r
}

func delContact(first *Contact, id int) *Contact {
	if first == nil {
		return first
	}

	for first != nil && first.Id == id {
		first = first.Next
	}

	res := first

	for first != nil{
		for first.Next != nil && first.Next.Id == id {
			first.Next = first.Next.Next
		}

		if first.Inner != nil {
			first.Inner = delContact(first.Inner, id)
		}

		first = first.Next
	}

	return res
}

func getContact(first *Contact, id int) *Contact{

	for first != nil {
		if first.Id == id {
			return first
		}

		if first.Inner != nil {
			inner := getContact(first.Inner, id)
			if inner != nil {
				return inner
			}
		}

		first = first.Next
	}

	return nil
}

func getNewId(first *Contact) int {
	if first == nil {
		return 1
	}

	r := 1

	for first != nil {

		if first.Id >= r {
			r = first.Id + 1
		}

		if first.Inner != nil {
			t := getNewId(first.Inner)
			if t >= r {
				r = t + 1
			}
		}

		first = first.Next
	}

	return r
}

func saveProfiles(){
	var list []Profile

	profiles.Range(func(key interface{}, value interface{}) bool{
		list = append(list, *value.(*Profile))
		return true
	})

	b, err := json.Marshal(list)
	if err == nil {
		f, err := os.Create(FILE_PROFILES + ".tmp")
		if err == nil {
			n, err := f.Write(b)
			if n == len(b) && err == nil {
				f.Close()

				os.Remove(FILE_PROFILES)
				os.Rename(FILE_PROFILES + ".tmp", FILE_PROFILES)
			} else {
				f.Close()
				logAdd(MESS_ERROR, "Не удалось сохранить профили: " + fmt.Sprint(err))
			}
		} else {
			logAdd(MESS_ERROR, "Не удалось сохранить профили: " + fmt.Sprint(err))
		}
	} else {
		logAdd(MESS_ERROR, "Не удалось сохранить профили: " + fmt.Sprint(err))
	}
}

func loadProfiles(){
	var list []Profile

	f, err := os.Open(FILE_PROFILES)
	defer f.Close()
	if err == nil {
		b, err := ioutil.ReadAll(f)
		if err == nil {
			err = json.Unmarshal(b, &list)
			if err == nil {
				for _, value := range list {
					profile := value
					profiles.Store(profile.Email, &profile)
				}
			} else {
				logAdd(MESS_ERROR, "Не получилось загрузить профили: " + fmt.Sprint(err))
			}
		} else {
			logAdd(MESS_ERROR, "Не получилось загрузить профили: " + fmt.Sprint(err))
		}
	} else {
		logAdd(MESS_ERROR, "Не получилось загрузить профили: " + fmt.Sprint(err))
	}
}

func saveOptions(){

	b, err := json.Marshal(options)
	if err == nil {
		f, err := os.Create(FILE_OPTIONS + ".tmp")
		if err == nil {
			n, err := f.Write(b)
			if n == len(b) && err == nil {
				f.Close()

				os.Remove(FILE_OPTIONS)
				os.Rename(FILE_OPTIONS + ".tmp", FILE_OPTIONS)
			} else {
				f.Close()
				logAdd(MESS_ERROR, "Не удалось сохранить настройки: " + fmt.Sprint(err))
			}
		} else {
			logAdd(MESS_ERROR, "Не удалось сохранить настройки: " + fmt.Sprint(err))
		}
	} else {
		logAdd(MESS_ERROR, "Не удалось сохранить настройки: " + fmt.Sprint(err))
	}
}

func loadOptions(){

	f, err := os.Open(FILE_OPTIONS)
	defer f.Close()
	if err == nil {
		b, err := ioutil.ReadAll(f)
		if err == nil {
			err = json.Unmarshal(b, &options)
			if err != nil {
				logAdd(MESS_ERROR, "Не получилось загрузить настройки: " + fmt.Sprint(err))
			}
		} else {
			logAdd(MESS_ERROR, "Не получилось загрузить настройки: " + fmt.Sprint(err))
		}
	} else {
		logAdd(MESS_ERROR, "Не получилось загрузить настройки: " + fmt.Sprint(err))
	}
}

func saveVNCList(){

	b, err := json.Marshal(array_vnc)
	if err == nil {
		f, err := os.Create(FILE_VNCLIST + ".tmp")
		if err == nil {
			n, err := f.Write(b)
			if n == len(b) && err == nil {
				f.Close()

				os.Remove(FILE_VNCLIST)
				os.Rename(FILE_VNCLIST + ".tmp", FILE_VNCLIST)
			} else {
				f.Close()
				logAdd(MESS_ERROR, "Не удалось сохранить список VNC: " + fmt.Sprint(err))
			}
		} else {
			logAdd(MESS_ERROR, "Не удалось сохранить список VNC: "+fmt.Sprint(err))
		}
	} else {
		logAdd(MESS_ERROR, "Не удалось сохранить список VNC: " + fmt.Sprint(err))
	}
}

func loadVNCList(){

	f, err := os.Open(FILE_VNCLIST)
	defer f.Close()
	if err == nil {
		b, err := ioutil.ReadAll(f)
		if err == nil {
			err = json.Unmarshal(b, &array_vnc)
			if err == nil {
				default_vnc = 0
				return
			} else {
				logAdd(MESS_ERROR, "Не получилось загрузить список VNC: " + fmt.Sprint(err))
			}
		} else {
			logAdd(MESS_ERROR, "Не получилось загрузить список VNC: " + fmt.Sprint(err))
		}
	} else {
		logAdd(MESS_ERROR, "Не получилось загрузить список VNC: " + fmt.Sprint(err))
	}
}

//пробежимся по профилям, найдем где есть контакты с нашим пид и добавим этот профиль нам
func addClientToProfile(client *Client) {
	profiles.Range(func (key interface {}, value interface {}) bool {
		profile := *value.(*Profile)
		if addClientToContacts(profile.Contacts, client, &profile) {
			//если мы есть хоть в одном конакте этого профиля, пробежимся по ним и отправим свой статус
			profile.clients.Range(func (key interface {}, value interface{}) bool {
				curClient := value.(*Client)
				sendMessage(curClient.Conn, TMESS_STATUS, cleanPid(client.Pid), "1")
				return true
			})
		}
		return true
	})
}

//пробежимся по всем контактам и если есть совпадение, то добавим ссылку на профиль этому клиенту
func addClientToContacts(contact *Contact, client *Client, profile *Profile) bool {
	res := false

	for contact != nil {
		if cleanPid(contact.Pid) == cleanPid(client.Pid) {
			client.profiles.Store(profile.Email, profile)
			res = true
		}

		if contact.Inner != nil {
			innerResult := addClientToContacts(contact.Inner, client, profile)
			if innerResult {
				res = true
			}
		}

		contact = contact.Next
	}

	return res
}

func checkStatuses(curClient *Client, first *Contact) {

	for first != nil {

		if first.Type != "fold" {
			_, exist := clients.Load(cleanPid(first.Pid))
			if exist {
				sendMessage(curClient.Conn, TMESS_STATUS, fmt.Sprint(cleanPid(first.Pid)), "1")
			} else {
				sendMessage(curClient.Conn, TMESS_STATUS, fmt.Sprint(cleanPid(first.Pid)), "0")
			}
		}

		if first.Inner != nil {

			checkStatuses(curClient, first.Inner)
		}

		first = first.Next
	}

}

func getInvisibleEmail(email string) string{

	len := len(email)
	if len > 10 {
		return email[:5] + "*****" + email[len - 5:]
	} else {
		return email[:1] + "*****" + email[len - 1:]
	}
}

func addCounter(bytes uint64) {
	counterData.mutex.Lock()
	defer counterData.mutex.Unlock()

	counterData.counterBytes[counterData.currentPos] = counterData.counterBytes[counterData.currentPos] + bytes
	counterData.counterConnect[counterData.currentPos] = counterData.counterConnect[counterData.currentPos] + 1
}

func swiftCounter() {
	counterData.mutex.Lock()
	defer counterData.mutex.Unlock()

	if time.Now().Hour() != counterData.currentPos {
		counterData.currentPos = time.Now().Hour()

		counterData.counterBytes[counterData.currentPos] = 0
		counterData.counterConnect[counterData.currentPos] = 0
	}
}

func cleanPid(pid string) string {
	//todo может потом стоит сюда добавить удаление и других символов
	return strings.Replace(pid, ":", "", -1)
}


//следующие функции нужны только для отладки
func printContact(node *Contact, tab int) {
	var t string

	for i := 0; i < tab; i++ {
		t = t + "\t";
	}

	fmt.Println(t, "id:\t", node.Id)
	fmt.Println(t, "type:\t", node.Type)
	fmt.Println(t, "capt:\t", node.Caption)
	fmt.Println(t, "pid:\t", node.Pid)
	fmt.Println(t, "digt:\t", node.Digest)
	fmt.Println(t, "salt:\t", node.Salt)
	fmt.Println(t, "next:\t", node.Next)
	fmt.Println(t, "inner:\t", node.Inner)
	fmt.Println()
}

func printContacts(first *Contact, tab int) {

	for first != nil {
		printContact(first, tab);
		if first.Inner != nil {
			printContacts(first.Inner, tab + 1);
		}
		first = first.Next
	}
}
