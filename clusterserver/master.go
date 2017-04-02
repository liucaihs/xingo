package clusterserver

import (
	"fmt"
	"github.com/viphxin/xingo/cluster"
	"github.com/viphxin/xingo/fserver"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"os"
	"os/signal"
	"sync"
)

type Master struct {
	OnlineNodes map[string]bool
	Cconf       *cluster.ClusterConf
	Childs      *cluster.ChildMgr
	sync.RWMutex
}

func NewMaster(path string) *Master {
	logger.SetPrefix("[MASTER]")
	cconf, err := cluster.NewClusterConf(path)
	if err != nil {
		panic("cluster conf error!!!")
	}
	GlobalMaster = &Master{
		OnlineNodes: make(map[string]bool),
		Cconf:       cconf,
		Childs:      cluster.NewChildMgr(),
	}
	//regest callback
	utils.GlobalObject.TcpPort = GlobalMaster.Cconf.Master.RootPort
	utils.GlobalObject.RpcCProtoc = cluster.NewRpcClientProtocol()
	utils.GlobalObject.RpcSProtoc = cluster.NewRpcServerProtocol()
	utils.GlobalObject.Protoc = utils.GlobalObject.RpcSProtoc
	utils.GlobalObject.OnClusterConnectioned = DoConnectionMade
	utils.GlobalObject.OnClusterClosed = DoConnectionLost
	utils.GlobalObject.Name = "master"
	if cconf.Master.Log != "" {
		utils.GlobalObject.LogName = cconf.Master.Log
		utils.ReSettingLog()
	}
	return GlobalMaster
}

func DoConnectionMade(fconn iface.Iconnection) {
	logger.Info("node connected to master!!!")
}

func DoConnectionLost(fconn iface.Iconnection) {
	logger.Info("node disconnected from master!!!")
	nodename, err := fconn.GetProperty("child")
	if err == nil {
		GlobalMaster.RemoveNode(nodename.(string))
	}
}

func (this *Master) WaitSignal() {
	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	logger.Info(fmt.Sprintf("signal catch: [%s]", sig))
}

func (this *Master) StartMaster() {
	s := fserver.NewServer()
	go utils.GlobalObject.RpcSProtoc.GetMsgHandle().StartWorkerLoop()
	s.Start()
	this.WaitSignal()
	s.Stop()
}

func (this *Master) AddRpcRouter(router interface{}) {
	//add rpc ---------------start
	if utils.GlobalObject.RpcSProtoc != nil {
		utils.GlobalObject.RpcSProtoc.AddRpcRouter(router)
	}
	if utils.GlobalObject.RpcCProtoc != nil {
		utils.GlobalObject.RpcCProtoc.AddRpcRouter(router)
	}
	//add rpc ---------------end
}

func (this *Master) AddNode(name string, writer iface.IWriter) {
	this.Lock()
	defer this.Unlock()

	this.Childs.AddChild(name, writer)
	writer.SetProperty("child", name)
	this.OnlineNodes[name] = true
}

func (this *Master) RemoveNode(name string) {
	this.Lock()
	defer this.Unlock()

	this.Childs.RemoveChild(name)
	delete(this.OnlineNodes, name)

}
