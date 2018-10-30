package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"strconv"

	"github.com/tiptok/gonat/global"
	"github.com/tiptok/gonat/model"

	"github.com/tiptok/gotransfer/comm"
	"github.com/tiptok/gotransfer/conn"
)

type protocol808 struct {
}

func (p protocol808) PacketMsg(obj interface{}) (data []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			debug.PrintStack()
			log.Printf("protocol808.PacketMsg panic recover! p: %v", p)
		}
	}()
	packdata, err := p.Packet(obj)
	if err != nil {
		return nil, err
	}

	if def, ok := obj.(model.IEntity); ok {
		entity := def.GetEntityBase()
		if packdata != nil && len(packdata) > 0 {
			log.Printf("MsgId:%X MsgBodyData:%s\n", def.GetMsgId().(int), comm.BinaryHelper.ToBCDString(packdata, 0, int32(len(packdata))))
		}
		buf := bytes.NewBuffer(nil)
		buf.Write(comm.BinaryHelper.Int16ToBytes(int16(def.GetMsgId().(int)))) //2消息Id
		buf.Write(comm.BinaryHelper.Int16ToBytes(int16(len(packdata))))        //2消息体属性
		simdata, _ := comm.BinaryHelper.GetBCDStringWL(comm.PaddingLeft(entity.SimNum, "0", 12), 6)
		buf.Write(simdata)                                             //12 手机卡号
		buf.Write(comm.BinaryHelper.Int16ToBytes(int16(entity.MsgSN))) //流水号
		//2消息包封装项
		buf.Write(packdata)
		buf.WriteByte(comm.BinaryHelper.GetCRC(buf.Bytes())) //计算crc
		return Byte808Enscape(buf.Bytes(), 0, buf.Len()), nil
	}
	return packdata, err
}

/*
	打包数据体
	obj 数据体
*/
func (p protocol808) Packet(obj interface{}) (packdata []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("protocol808.Packet panic recover! p: %v", p)
		}
	}()
	if def, ok := obj.(model.IEntity); ok {
		entity := def.GetEntityBase()
		var sMethodName string
		var cmdCode interface{}
		if entity.SubMsgId == nil {
			cmdCode = def.GetMsgId()
		} else {
			cmdCode = entity.SubMsgId
		}
		icmdCode, err := strconv.Atoi(fmt.Sprintf("%v", cmdCode))
		if err != nil {
			return nil, err
		}
		sMethodName = GetFuncName(icmdCode)
		log.Printf("InvokeFunc:%s\n", sMethodName)
		value, err := comm.ParseHelper.InvokeFunc(&JTB808Packer{}, sMethodName, obj)
		if err == nil {
			packdata = (value[0].Interface()).([]byte)
		} else {
			return nil, err
		}
	} else {
		err = errors.New("非809实体,发送异常")
	}
	return packdata, err
}

/*
	分包处理
	packdata 解析出一个完整包
	leftdata 解析剩余报文的数据
	err 	 分包错误
*/
func (p protocol808) ParseMsg(data []byte, c *conn.Connector) (packdata [][]byte, leftdata []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("protocol808.ParseMsg panic recover! p: %v", p)
		}
	}()
	packdata, leftdata, err = comm.ParseHelper.ParsePartWithPool(data, 0x7E, 0x7E, c.Pool)
	return packdata, leftdata, err
}

/*
	解析数据
	obj 解析出对应得数据结构
	err 解析出错
*/
func (p protocol808) Parse(packdata []byte) (obj interface{}, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("protocol808.Parse panic recover! p: %v", p)
			debug.PrintStack()
		}
	}()
	//转义
	data, err := Byte808Descape(packdata, 0, len(packdata))

	global.Debug(global.F(global.TCP, global.SVR809, "收到数据: %v"), hex.EncodeToString(packdata))
	//global.Debug(global.F(global.TCP, global.SVR809, "收到数据: %v"), hex.EncodeToString(data))
	//CRC Check
	tmpCrc := data[len(data)-1]
	checkCrc := comm.BinaryHelper.GetCRC(data[:len(data)-1])
	if checkCrc != tmpCrc {
		err = errors.New(fmt.Sprintf("CRC CHECK Error->%x != %x %v", uint16(tmpCrc), uint16(checkCrc), hex.EncodeToString(packdata)))
		return nil, err
	}
	h := model.EntityBase{}
	//数据头
	h.MsgId = comm.BinaryHelper.ToUInt16(data, 0)
	headerProperty := comm.BinaryHelper.ToInt16(data, 2)
	msgBodyLenght := headerProperty & 0x3FF
	bPartMsg := (headerProperty & 0x2000) > 0
	h.SimNum = comm.BinaryHelper.ToBCDString(data, 4, 6)
	h.MsgSN = int(comm.BinaryHelper.ToInt16(data, 10))
	//消息包分装
	sMethodName := fmt.Sprintf("J%s", comm.BinaryHelper.ToBCDString(data, 0, 2))
	var msgBody []byte
	if bPartMsg {
		h.PackageCount = int(comm.BinaryHelper.ToInt16(data, 12))
		h.PackageSN = int(comm.BinaryHelper.ToInt16(data, 14))
		msgBody = data[16 : 16+msgBodyLenght] //16+msgBodyLenght
	} else {
		msgBody = data[12 : 12+msgBodyLenght]
	}
	log.Println(msgBodyLenght, sMethodName, comm.BinaryHelper.ToBCDString(msgBody, 0, int32(len(msgBody))))
	value, err := comm.ParseHelper.InvokeFunc(&JTB808Parse{}, sMethodName, msgBody, h)
	if err != nil {
		return nil, err
	}
	if value != nil && len(value) > 0 {
		obj = value[0].Interface()
	}

	return obj, err
}

func Byte808Descape(value []byte, startIndex int, length int) ([]byte, error) {
	ilength := len(value)
	if (startIndex + length) > ilength {
		return nil, errors.New("Byte809Descape 长度不足，下标越界")
	}
	buf := new(bytes.Buffer)
	/*去头去尾*/
	for i := startIndex + 1; i < ilength-1; i++ {
		if value[i] == 0x7D {
			if value[i+1] == 0x01 {
				buf.WriteByte(0x7D)
			} else if value[i+1] == 0x02 {
				buf.WriteByte(0x7E)
			}
			i++
		} else {
			if i != ilength-1 {
				buf.WriteByte(value[i])
			}
		}

	}
	return buf.Bytes(), nil
}

func Byte808Enscape(value []byte, startIndex int, length int) []byte {
	ilength := len(value)
	if (startIndex + length) > ilength {

	}
	buf := new(bytes.Buffer)
	buf.WriteByte(0x7E)
	for i := startIndex; i < ilength; i++ {
		if value[i] == 0x7D {
			buf.WriteByte(0x7D)
			buf.WriteByte(0x01)
		} else if value[i] == 0x7E {
			buf.WriteByte(0x7D)
			buf.WriteByte(0x02)
		} else {
			buf.WriteByte(value[i])
		}
	}
	buf.WriteByte(0x7E)
	return buf.Bytes()
}

func GetFuncName(icmd int) string {
	sFuncName := fmt.Sprintf("%x", icmd)
	iLen := len(sFuncName)
	if iLen < 4 {
		for i := 0; i < (4 - iLen); i++ {
			sFuncName = fmt.Sprintf("%s%s", "0", sFuncName)
		}
	}
	return fmt.Sprintf("%s%s", "J", sFuncName)
}
