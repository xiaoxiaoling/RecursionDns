package main

import (
	"godns/models"

	"github.com/astaxie/beego"
)

func main() {
	models.MInit()
	go models.ListenAndServe()
	for i := 0; i < 1; i++ {
		go models.Worker()
	}
	beego.Run()
}
