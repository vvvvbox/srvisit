package main

import (
	"runtime"
	"fmt"
)



func main(){
	logAdd(MESS_INFO, "Запускается сервер reVisit версии " + REVISIT_VERSION)

	runtime.GOMAXPROCS(runtime.NumCPU())

	loadVNCList()
	loadOptions()
	loadProfiles()

	go helperThread() //используем для периодических действий(сохранения и т.п.)
	go httpServer()
	go mainServer()
	go dataServer()

	var r string
	for r != "quit" {
		fmt.Scanln(&r)
	}

	logAdd(MESS_INFO, "Завершили работу")
}
