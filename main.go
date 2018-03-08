package main

import (
	"runtime"
	"fmt"
)

func main(){
	logAdd(MESS_INFO, "Запускается сервер reVisit версии " + REVISIT_VERSION)

	runtime.GOMAXPROCS(runtime.NumCPU())

	loadVNCList()
	loadCounters()
	loadOptions()
	loadProfiles()

	//go finderNeighbours()	//поиск соседей
	go helperThread() 		//используем для периодических действий(сохранения и т.п.)
	go httpServer()			//обработка веб запросов
	go mainServer()			//обработка основных команд от клиентов
	go dataServer()			//обработка потоков данных от клиентов

	var r string
	for r != "quit" {
		fmt.Scanln(&r)
	}

	logAdd(MESS_INFO, "Завершили работу")
}
