package main

import (
	"fauth/pb/login2chk"
	"fauth/pb/msghead"
	"fauth/tcpServer/zero"
	"fauth/utils/dbutils"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq4"
	"gopkg.in/mgo.v2"
)

var MgoSess *mgo.Session //MGPool mgopool.Pool

const (
	DB_PLAYER string = "PlayerDB_DWC"
	DB_LOG    string = "LogDB_DWC"
)

type RpcCtrl struct {
	ListenAddr1 string
	Listener1   net.Listener
	ListenAddr2 string
	Listener2   net.Listener
	AuthAddr    string
	AuthSock    *zmq.Socket
	OrderAddr   string
	OrderSock   *zmq.Socket
	JavaAddr    string
	JavaSock    *zmq.Socket
	PushAddr    string
	PushSock    *zmq.Socket
}

func main() {
	flag.Parse()
	Num := flag.NArg()
	if Num != 7 {
		fmt.Printf("params is %d,not 7\n", Num)
		return

	}
	/*
		listen1 := flag.Arg(0)   //0.0.0.0:7799 tcp
		authAddr := flag.Arg(1)  //tcp:127.0.0.1:5569
		pushAddr := flag.Arg(2)  //tcp:127.0.0.1:7783
		listen2 := flag.Arg(3)   //0.0.0.0;8033 websock
		javaAddr := flag.Arg(4)  //tcp://127.0.0.1:5689
		orderAddr := flag.Arg(5) //tcp://127.0.0.1:5539

		//hostAddr := flag.Arg(0) //:5570
		//dbAddr := flag.Arg(1)
		//dbUser := flag.Arg(2)
		//dbPwd := flag.Arg(3)

		//hostAddr := "192.168.8.33:7799"
		authSock, _ := zmq.NewSocket(zmq.DEALER)
		authSock.Connect(authAddr)

		javaSock, _ := zmq.NewSocket(zmq.DEALER)
		javaSock.Connect(javaAddr)

		orderSock, _ := zmq.NewSocket(zmq.DEALER)
		orderSock.Connect(orderAddr)
		pushSock, _ := zmq.NewSocket(zmq.PUSH)
		pushSock.Connect(pushAddr)
	*/
	ss, err := zero.NewSocketService("0.0.0.0:7799", 5*time.Second, 30*time.Second)
	if err != nil {
		return
	}

	//set Heartbeat
	//ss.SetHeartBeat(5*time.Second, 30*time.Second)

	ss.RegMessageHandler(OnMessage)
	ss.RegConnectHandler(OnConnect)
	ss.RegDisconnectHandler(OnDisconnect)

	MgoSess = dbutils.NewDBSession("127.0.0.1:27017", "test", "123456")
	MgoSess.SetPoolLimit(5)
	ss.Run()
}

func OnMessage(ss *zero.Session, msg *zero.Message) {
	fmt.Printf("receive msgID:%d, msgSize:%d\n", msg.GetID(), msg.GetSize())

	msgId := msghead.EmsgType(msg.GetID())
	if msgId == msghead.EmsgType_e_clt2login_login {
		userLogin := &login2chk.UserLogin{}
		if err := proto.Unmarshal(msg.GetData(), userLogin); err == nil {
			mogoSn := MgoSess.Copy()
			defer mogoSn.Close()
			coll := mogoSn.DB(DB_LOG).C("clt2login_login")
			if err := coll.Insert(userLogin); err != nil {
				//log.Info("insert log errored !", err.Error())
			}
			//log.Info("acc:%s ,login_time:%s", userLogin.GetAcc(), time.Unix(time.Now().Unix(), 0).Format("2006-01-02 :15:03:04"))
		} else {

		}
	}

}

func OnDisconnect(s *zero.Session, err error) {
	fmt.Println(s.GetConn().GetName() + " disconnected.")
}

func OnConnect(s *zero.Session) {
	fmt.Println(s.GetConn().GetName() + " connected.")
}
