package main

import (
	"log"

	"github.com/astaxie/beego/config"
)

var Param *Params

//Params config item
type Params struct {
	/*******基本参数*******/
	RemoteIP           string
	RemotePort         int
	SimNum             string
	InPressureTestMode bool  //是否在压力测试模式
	TermNum            int   //终端接入数
	OriginSimNum       int64 //起始SimNum编号 后面的终端为 OriginSimNum+ SN
	PosInterval        int   //刷新间隔
}

func ConfigInit() {
	Param = &Params{}
	Param.LoadConfig("ini", "param.conf")
}

//LoadConfig load config
func (p *Params) LoadConfig(pType string, fName string) *Params {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("LoadConfig 读取配置异常r! p: %v", rec)
		}
	}()
	con, err := config.NewConfig(pType, fName)
	if err != nil {
		log.Printf("LoadConfig 加载配置异常 e: %v", err)
	}
	p.RemoteIP = con.String("RemoteIP")
	p.RemotePort, _ = con.Int("RemotePort")
	p.SimNum = con.String("SimNum")
	p.InPressureTestMode, _ = con.Bool("InPressureTestMode")
	p.TermNum, _ = con.Int("TermNum")
	p.OriginSimNum, _ = con.Int64("OriginSimNum")
	p.PosInterval, _ = con.Int("PosInterval")
	log.Printf("Load Config:%v\n", *p)
	return p
}
