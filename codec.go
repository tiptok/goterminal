package main

import (
	"bytes"

	"github.com/tiptok/gonat/model"
	"github.com/tiptok/gonat/model/jtb808/down"
	"github.com/tiptok/gonat/model/jtb808/up"
	"github.com/tiptok/gotransfer/comm"
)

type JTB808Packer struct {
}

/*
   J0001 终端通用应答0x0001
*/
func (p *JTB808Packer) J0001(obj interface{}) (packdata []byte, err error) {
	inEntity := obj.(*up.TermCommonReply)
	buf := bytes.NewBuffer(nil)
	buf.Write(comm.BinaryHelper.Int16ToBytes(int16(inEntity.RspMsgSN)))
	buf.Write(comm.BinaryHelper.Int16ToBytes(int16(inEntity.RspMsgId)))
	buf.WriteByte(inEntity.RspResult)
	return buf.Bytes(), nil
}

/*
   J1002 主链路登录应答
*/
func (p *JTB808Packer) J0002(obj interface{}) (packdata []byte, err error) {
	buf := bytes.NewBuffer(nil)
	return buf.Bytes(), nil
}

/*
   J0102 终端鉴权
*/
func (p *JTB808Packer) J0102(obj interface{}) (packdata []byte, err error) {
	inEntity := obj.(*up.TermAuth)
	buf := bytes.NewBuffer(nil)
	buf.WriteString(inEntity.AuthCode)
	return buf.Bytes(), nil
}

/*
   J0801 终端升级结果通知
*/
func (p *JTB808Packer) J0108(obj interface{}) (packdata []byte, err error) {
	inEntity := obj.(*up.UpgradeResult)
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(inEntity.UpgradeType)
	buf.WriteByte(inEntity.Result)
	return buf.Bytes(), nil
}

/*
   J8003 补传分包请求 0x8003
*/
func (p *JTB808Packer) J8003(obj interface{}) (packdata []byte, err error) {
	inEntity := obj.(*up.PacketReSendRequest)
	buf := bytes.NewBuffer(nil)
	buf.Write(comm.BinaryHelper.Int16ToBytes(int16(inEntity.OrginalOrderID)))
	if inEntity.PackageSNList != nil {
		for i := 0; i < len(inEntity.PackageSNList); i++ {
			buf.Write(comm.BinaryHelper.Int16ToBytes(inEntity.PackageSNList[i]))
		}
	}
	return buf.Bytes(), nil
}

/*
   J0200 位置信息汇报
*/
func (p *JTB808Packer) J0200(obj interface{}) (packdata []byte, err error) {
	buf := bytes.NewBuffer(nil)
	inEntity := obj.(*model.TermPosition)
	buf.Write(comm.BinaryHelper.UInt32ToBytes(uint(inEntity.AlarmFlag)))
	buf.Write(comm.BinaryHelper.UInt32ToBytes(uint(inEntity.StateFlag)))
	buf.Write(comm.BinaryHelper.UInt32ToBytes(uint(inEntity.Lat * 1000000)))
	buf.Write(comm.BinaryHelper.UInt32ToBytes(uint(inEntity.Lon * 1000000)))
	buf.Write(comm.BinaryHelper.UInt16ToBytes(uint16(inEntity.Altitude)))
	buf.Write(comm.BinaryHelper.UInt16ToBytes(uint16(inEntity.Speed)))
	buf.Write(comm.BinaryHelper.UInt16ToBytes(uint16(inEntity.Direction)))
	gps, _ := comm.BinaryHelper.GetBCDString(inEntity.GpsTime.Format("060102150405"))
	buf.Write(gps)
	return buf.Bytes(), nil
}

type JTB808Parse struct {
}

//J8001 0x8001 平台通用应答
func (p *JTB808Parse) J8001(msgBody []byte, h model.EntityBase) interface{} {
	outEntity := &down.PlatformCommonReply{}
	outEntity.SetEntity(h)
	outEntity.RspMsgSN = int(comm.BinaryHelper.ToInt16(msgBody, 0)) //应答ID
	outEntity.RspMsgId = int(comm.BinaryHelper.ToInt16(msgBody, 2))
	outEntity.RspResult = msgBody[4]

	/*是否需要应答*/
	// outEntity.IsNeedRsp = false
	// outEntity.DownRspEntity = nil
	return outEntity
}

//J8102 0x8102 平下发终端升级包
func (p *JTB808Parse) J8108(msgBody []byte, h model.EntityBase) interface{} {
	outEntity := &down.SendUpgradePackage{}
	outEntity.SetEntity(h)
	if h.PackageSN == 1 {
		outEntity.UpgradeType = msgBody[0] //应答ID
		outEntity.ManufacturerId = comm.BinaryHelper.ToASCIIString(msgBody, 1, 5)
		SoftwareVersionLen := msgBody[6]
		outEntity.SoftwareVersion = comm.BinaryHelper.ToASCIIString(msgBody, 7, int32(SoftwareVersionLen))
		outEntity.FileLength = int(comm.BinaryHelper.ToInt32(msgBody, int32(7+SoftwareVersionLen)))
		//outEntity.DataPackage = msgBody[11+SoftwareVersionLen:]
	} else {
		//outEntity.DataPackage = msgBody[:]
	}

	/*是否需要应答*/
	// outEntity.IsNeedRsp = false
	// outEntity.DownRspEntity = nil
	return outEntity
}
