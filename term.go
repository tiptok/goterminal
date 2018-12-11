package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/tiptok/gonat/model"
	"github.com/tiptok/gonat/model/jtb808/down"
	"github.com/tiptok/gonat/model/jtb808/up"
	"github.com/tiptok/gotransfer/conn"
)

type Terminal struct {
	Client    *conn.TcpClient
	Regist    int
	Auth      int
	OrderId   int32
	AuthCode  string
	SimNum    string
	close     chan struct{}
	sendQueue chan interface{}
}

func (t *Terminal) GetOrderId() int {
	if t.OrderId >= math.MaxInt32 || t.OrderId < 0 {
		atomic.SwapInt32(&t.OrderId, 1)
	}
	return int(atomic.AddInt32(&t.OrderId, 1))
}

func (t *Terminal) SendEntity(e interface{}) ([]byte, error) {
	if !t.Client.Conn.IsConneted {
		return nil, fmt.Errorf("无连接到服务器")
	}
	send, err := conn.SendEntity(e, t.Client.Conn)
	log.Println("Send Data:", hex.EncodeToString(send))
	return send, err
}

//KeepConn 维持链路信息 数据 发送
func (t *Terminal) KeepConn() {
	for {
		select {
		case <-t.close:
			return
		case send := <-t.sendQueue:
			if send != nil {
				_, err := t.SendEntity(send)
				if err != nil {
					log.Println(err.Error())
				}
			}
		case <-time.After(time.Second * 15):
			// if t.Regist == 0 {
			// 	/*sendRegist*/

			// }
			// if t.Regist == 1 && t.Auth == 0 {
			// 	/*send auth*/
			// }
			// if !t.Client.Conn.IsConneted {
			// 	bRestart := t.Client.ReStart()
			// 	log.Printf("Term:%s Restart Result:%v \n", t.SimNum, bRestart)
			// }
			heart := &up.TermHeartbeat{
				EntityBase: model.EntityBase{
					SimNum: t.SimNum,
					MsgSN:  t.GetOrderId(),
				},
			}

			_, err := t.SendEntity(heart)
			if err != nil {
				log.Println(err.Error())
			} else {
				if t.Auth != 1 {
					auth := &up.TermAuth{
						EntityBase: model.EntityBase{
							SimNum: t.SimNum,
							MsgSN:  t.GetOrderId(),
						},
						AuthCode: "123456",
					}
					_, err = t.SendEntity(auth)
					if err != nil {
						log.Println(err.Error())
					}
				}
				t.Auth = 1
			}
			/*发送心跳*/
		}
	}
}

//PosInterval  定时发送定位数据
func (t *Terminal) PosInterval(interval int) {
	//rand.Seed(255)
	for {
		select {
		case <-t.close:
			return
		case <-time.After(time.Duration(interval) * time.Millisecond):
			if t.Auth == 1 {
				pos := &model.TermPosition{
					EntityBase: model.EntityBase{
						SimNum: t.SimNum,
						MsgSN:  t.GetOrderId(),
					},
					GpsTime: time.Now(),
					Lon:     121.123456,
					Lat:     35.356789,
					// AlarmFlag: int64(rand.Int()),
					// StateFlag: int64(rand.Int()),
					Speed: 60,
				}
				t.sendQueue <- pos
			}
		}
	}
}

func NewTerminal(sIP string, iPort int, sSimNum string) *Terminal {
	t := &Terminal{
		Client: &conn.TcpClient{
			P: &protocol808{},
		},
		SimNum:    sSimNum, //Param.SimNum
		sendQueue: make(chan interface{}, 100),
	}

	t.Client.NewTcpClient(sIP, iPort, 500, 500)
	t.Client.Start(&TermHandler{Term: t}) //

	return t
}

//TermHandler 终端处理
type TermHandler struct {
	Term    *Terminal
	Upgrage *TerminalUpgrageInfo
}

//OnConnect 连接事件
func (cli *TermHandler) OnConnect(c *conn.Connector) bool {
	log.Println("client On connect", c.RemoteAddress)
	return true
}

//OnReceive 接收事件
func (cli *TermHandler) OnReceive(c *conn.Connector, d conn.TcpData) bool {
	log.Printf("Rece Data:%v\n", hex.EncodeToString(d.Bytes()))
	obj, err := c.ParseToEntity(d.Bytes())
	if err != nil {
		log.Println("OnReceive Error.", err.Error())
	}
	if obj != nil {
		log.Println("接收到实体:", obj)
	}
	//var ientity model.IEntity //应答实体
	if entity, ok := obj.(model.IEntity); ok {
		entityBase := entity.GetEntityBase()
		cmdcode := entityBase.MsgId.(uint16)
		switch cmdcode {
		case model.JPlatformCommonReply:
			//platFormCommRepley := (entity).(*down.PlatformCommonReply)
			//log.Println(platFormCommRepley)
		case model.JSendUpgradePackage:
			upgradePackage := (entity).(*down.SendUpgradePackage)
			commonReply := &up.TermCommonReply{
				EntityBase: model.EntityBase{
					SimNum: entityBase.SimNum,
					MsgSN:  cli.Term.GetOrderId(),
				},
				RspMsgSN:  entityBase.PackageSN,
				RspMsgId:  upgradePackage.GetMsgId().(int),
				RspResult: 0,
			}
			cli.Term.SendEntity(commonReply)
			log.Printf("终端升级包 SimNum:%s 包序:%d 总包数:%d 消息ID:%x\n", commonReply.SimNum, entityBase.PackageSN, entityBase.PackageCount, commonReply.RspMsgId)
			if cli.Upgrage == nil && entityBase.PackageSN == 1 {
				cli.Upgrage = NewTerminalUpgrageInfo(upgradePackage.FileLength)
				log.Printf("新增终端升级 版本:%s 升级包长度:%d 总分包数:%d", upgradePackage.SoftwareVersion, upgradePackage.FileLength, entityBase.PackageCount)
			}
			if cli.Upgrage != nil {
				cli.Upgrage.UpdateUpgrage(entityBase.PackageSN, upgradePackage.DataPackage)
				if entityBase.PackageSN == entityBase.PackageCount {
					/*升级完成*/
					log.Printf("终端升级完成 长度:%d", cli.Upgrage.Length)
					cli.Upgrage = nil //置空

					/*发送升级完成通知*/
					upgrageResult := &up.UpgradeResult{
						EntityBase: model.EntityBase{
							SimNum: entityBase.SimNum,
							MsgSN:  cli.Term.GetOrderId(),
						},
						UpgradeType: 0,
						Result:      0,
					}
					cli.Term.SendEntity(upgrageResult)
				}
			}
		}
	}
	return true
}

//OnClose 关闭事件
func (cli *TermHandler) OnClose(c *conn.Connector) {
	//从列表移除
	log.Println("Client OnClose:", c.RemoteAddress)
}

//TerminalUpgrageInfo 终端升级详细信息
type TerminalUpgrageInfo struct {
	LastPackageSN int32
	UpgrageData   []byte
	Length        int   //升级文件长度
	CurrentIdx    int32 //当前下标
}

//NewTerminalUpgrageInfo 新建终端升级信息
func NewTerminalUpgrageInfo(length int) *TerminalUpgrageInfo {
	return &TerminalUpgrageInfo{
		LastPackageSN: 0,
		UpgrageData:   make([]byte, length),
		Length:        length,
		CurrentIdx:    0,
	}
}

//UpdateUpgrage  更新终端升级信息
func (upgrage *TerminalUpgrageInfo) UpdateUpgrage(curSN int, data []byte) {
	if curSN == (int(upgrage.LastPackageSN) + 1) {
		copy(upgrage.UpgrageData, data)
		atomic.AddInt32(&upgrage.CurrentIdx, int32(len(data)))
	}
	atomic.StoreInt32(&upgrage.LastPackageSN, int32(curSN))
}
