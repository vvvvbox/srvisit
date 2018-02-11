package main

import (
	"runtime"
	"strconv"
	"fmt"
)

func main(){
	logAdd(MESS_INFO, "Запускается сервер reVisit версии " + strconv.FormatFloat(REVISIT_VERSION, 'f', -1, 64))

	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu)

	loadProfiles()

	go helperThread() //используем для периодических действий(сохранения и т.п.)
	go httpServer()
	go mainServer()
	go dataServer()

	fmt.Scanln()
}
