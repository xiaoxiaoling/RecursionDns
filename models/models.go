package models

import (
	"database/sql"
	"flag"
	"fmt"
	"net"
	//"os"
	//"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	_ "github.com/go-sql-driver/mysql"
)

type DBUnit struct {
	id     int
	nsg_id int
	domain string
}

// 请求集
type ReqsMap struct {
	_lock sync.RWMutex
	reqs  map[int]DBUnit
}

// 结果集
type NSRecs struct {
	_lock  sync.RWMutex
	ns_map map[string][]string
}

// db的 ns集
type DBns struct {
	_lock  sync.RWMutex
	ns_map map[int][]string
}

// 临时存储请求集
type TmpReqs struct {
	_lock  sync.RWMutex
	ns_map map[string][]int
}

type RDdata struct {
	domain, value string
	_type         int
}

type SqlModels struct {
	Ip      string
	Port    string
	Login   string
	Passwd  string
	LibName string
}

func getVerNS(id int) (res []string) {

	NSmap._lock.RLock()
	defer NSmap._lock.RUnlock()
	res = NSmap.ns_map[id]
	return
}

type RootTimeOffset struct {
	_lock  sync.RWMutex
	to_map map[string]int64
}

func RootSleep(qname string) {
	TimeOffs._lock.Lock()
	defer TimeOffs._lock.Unlock()

	if v, ok := TimeOffs.to_map[qname]; ok {
		if time.Now().UnixNano()-v < 10*1000*1000 {
			time.Sleep(10 * time.Millisecond)
		}
		TimeOffs.to_map[qname] = time.Now().UnixNano()

	}
}

// 统一错误处理
func checkErr(err error) {
	if err != nil {
		// panic(err)
		Log.Error(err.Error())
	}
}

var (
	Server     ServerMux
	mobiledb   *sql.DB = nil
	destdb     *sql.DB = nil
	NSmap      DBns
	RNPool     sync.Pool
	RNChan     chan RequestNode
	ResultRecs NSRecs
	RootCron   map[string]*uint32
	TimeOffs   RootTimeOffset
	CheckFlag  uint32
	Log        *logs.BeeLogger
)

const (
	FRAGMENT int = 100000
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func MInit() {
	t0 := time.Now()
	fn := "log//" + "API_" + strconv.Itoa(t0.Year()) + "_" + t0.Month().String() + "_" + strconv.Itoa(t0.Day()) + "_" + strconv.Itoa(t0.Hour()) + "_" + strconv.Itoa(t0.Minute()) + "_" + strconv.Itoa(t0.Second())
	Log = logs.NewLogger(10000)

	// Log.SetLogger("console", `{"level":1}`)
	Log.SetLogger("console", ``)
	Log.SetLogger("file", `{"filename":"`+fn+".Log"+`"}`)
	Log.EnableFuncCallDepth(true)

	RootCron = make(map[string]*uint32)
	var conInfo SqlModels

	conInfo.Ip = beego.AppConfig.String("SDB_IP")
	conInfo.Port = beego.AppConfig.String("SDB_Port")
	conInfo.LibName = beego.AppConfig.String("SDB_LibName")
	conInfo.Login = beego.AppConfig.String("SDB_Login")
	conInfo.Passwd = beego.AppConfig.String("SDB_Passwd")

	//mobiledb = DB_Open(conInfo)
	//mobiledb.SetMaxOpenConns(200)
	//mobiledb.SetMaxIdleConns(100)
	//mobiledb.SetConnMaxLifetime(0)
	conInfo.Ip = beego.AppConfig.String("DDB_IP")
	conInfo.Port = beego.AppConfig.String("DDB_Port")
	conInfo.LibName = beego.AppConfig.String("DDB_LibName")
	conInfo.Login = beego.AppConfig.String("DDB_Login")
	conInfo.Passwd = beego.AppConfig.String("DDB_Passwd")

	//destdb = DB_Open(conInfo)
	//destdb.SetMaxOpenConns(200)
	//destdb.SetMaxIdleConns(100)
	//destdb.SetConnMaxLifetime(0)
	InitDS()
	LoadConfig()
	//sql_getRootDomain()
	InitSets()
	//getNSs()
	Server.InitServer(4)

	ReqA = make(map[string][]RequestNode)
	ReqNS = make(map[string][]RequestNode)
	ReqAAAA = make(map[string][]RequestNode)
	ReqMX = make(map[string][]RequestNode)
	RNPool.New = func() interface{} {
		var rn RequestNode
		return rn
	}
	RNChan = make(chan RequestNode, 100)
	/*go func() {
		for {

			<-time.After(time.Duration(1) * time.Second)
			ReqsLock.Lock()

			for k1, v := range ReqA {
				ish := false
				for _, v2 := range v {
					if time.Now().Unix()-v2.TPoint > 5 {
						ish = true
						break
					}
				}
				if ish {
					delete(ReqA, k1)
				}
			}
			for k1, v := range ReqNS {
				ish := false
				for _, v2 := range v {
					if time.Now().Unix()-v2.TPoint > 5 {
						ish = true
						break
					}
				}
				if ish {
					delete(ReqNS, k1)
				}
			}
			for k, v := range ReqAAAA {
				ish := false
				for _, v2 := range v {
					if time.Now().Unix()-v2.TPoint > 5 {
						ish = true
						break
					}
				}
				if ish {
					delete(ReqAAAA, k)
				}
			}
			for k1, v := range ReqMX {
				ish := false
				for _, v2 := range v {
					if time.Now().Unix()-v2.TPoint > 5 {
						ish = true
						break
					}
				}
				if ish {
					delete(ReqMX, k1)
				}
			}
			ReqsLock.Unlock()
		}
	}()*/

}

func InitSets() {
	TimeOffs.to_map = make(map[string]int64)
}

func DB_Open(conInfo SqlModels) *sql.DB {
	//db, err := sql.Open("mysql", "sgxz:4399feicaiteam@/xvz2_mobile")

	constr := conInfo.Login + ":" + conInfo.Passwd + "@tcp(" + conInfo.Ip + ":" +
		conInfo.Port + ")/" + conInfo.LibName

	db, err := sql.Open("mysql", constr)
	checkErr(err)
	return db
}

func DB_Close(db *sql.DB) {
	if db != nil {
		db.Close()
	}
}
func getMaxId(tb string) (id int) {

	rows, err := mobiledb.Query("SELECT MAX(domainsID) from " + tb)
	//rows, err := mobiledb.Query("SELECT MAX(id) from " + tb)
	defer rows.Close()
	checkErr(err)

	if rows.Next() {
		rows.Scan(&id)
		//checkErr(err)
	}
	return
}

func getList(min, max int, tb string) (reqs map[int]DBUnit) {
	Log.Debug("mobiledb.Stats() %v ", mobiledb.Stats())
	reqs = make(map[int]DBUnit)
	rows, err := mobiledb.Query("SELECT domainsID,tdomain,nsGroupID FROM " + tb + " WHERE healthCheckDate BETWEEN '2016-01-01 00:01:01' and '2016-07-01 00:01:01' and  healthStatus != 0 and domainsID >= " + strconv.Itoa(min) + " and domainsID < " + strconv.Itoa(max))
	//rows, err := mobiledb.Query("SELECT id,tdomain,nsg_id FROM " + tb + " WHERE  id >= " + strconv.Itoa(min) + " and id < " + strconv.Itoa(max))
	defer rows.Close()
	checkErr(err)

	var ru DBUnit
	for rows.Next() {
		err = rows.Scan(&ru.id, &ru.domain, &ru.nsg_id)
		checkErr(err)

		if ru.id > 0 {
			reqs[ru.id] = ru
		}

	}
	return
}

func getTables() (tbs []string) {
	rows, _ := mobiledb.Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES  WHERE TABLE_NAME REGEXP 'domain_[0-9]'")
	defer rows.Close()
	//checkErr(err)

	var str string
	for rows.Next() {
		_ = rows.Scan(&str)
		//checkErr(err)

		if len(str) > 0 {
			tbs = append(tbs, str)
		}

	}
	return

}

func getDestMaxId(tb string) (id int) {

	rows, _ := destdb.Query("SELECT MAX(id) from " + tb)
	defer rows.Close()
	//checkErr(err)

	if rows.Next() {
		rows.Scan(&id)
		//checkErr(err)
	}
	return
}
func getDestList(min, max int, tb string) (reqs map[int]DBUnit) {
	reqs = make(map[int]DBUnit)
	rows, _ := destdb.Query("SELECT id,domain,nsg_id FROM " + tb + " WHERE state = 0 and LENGTH(ns)=0  and id >= " + strconv.Itoa(min) + " and id < " + strconv.Itoa(max))
	defer rows.Close()
	//checkErr(err)

	var ru DBUnit
	for rows.Next() {
		_ = rows.Scan(&ru.id, &ru.domain, &ru.nsg_id)
		//checkErr(err)

		if ru.id > 0 {
			reqs[ru.id] = ru
		}

	}
	return
}

func getNSs() {
	rows, err := mobiledb.Query("SELECT ns,groupID FROM dns_servers_groups")
	//rows, err := mobiledb.Query("SELECT ns,nsg_id FROM domain_ns")
	defer rows.Close()
	checkErr(err)
	var id int
	var str string
	for rows.Next() {
		err = rows.Scan(&str, &id)
		checkErr(err)
		strs := strings.Split(str, " ")
		for _, v := range strs {
			NSmap.ns_map[id] = append(NSmap.ns_map[id], Fqdn(v))
		}

	}
	return
}

func sql_setRootDomain(domain, value string, _type uint16) {

	//var res string
	tablename := "root_domain"
	cmdstr := "replace into " + tablename + " (domain,type,value) " + "values (?,?,?)"
	stmt, _ := destdb.Prepare(cmdstr)
	defer stmt.Close()
	_, err := stmt.Exec(domain, _type, value)
	checkErr(err)

}

func sql_getRootDomain() {
	tablename := "root_domain"
	rows, err := destdb.Query("SELECT domain,type,value FROM " + tablename)
	defer rows.Close()
	checkErr(err)
	var domain, val string
	var _type uint16

	for rows.Next() {
		rows.Scan(&domain, &_type, &val)
		Log.Debug("sql_getRootDomain  %v  %v  %v", domain, _type, val)
		fmt.Println(domain, _type, val)
		switch _type {
		case TypeA:
			AddA_str(domain, val)
		case TypeNS:
			AddNS(domain, val)
		case TypeCNAME:
			SetOriCNAME(domain, val)
		case TypeAAAA:
			AddAAAA_str(domain, val)
		}
	}
}

func IsVerified(reqns []string, ns []string) (isv bool) {
	for _, v := range reqns {
		for _, v2 := range ns {
			if v == v2 {
				isv = true
				return
			}
		}
	}

	return
}

func GetNSs(qname string, resp *[]string) {
	qns := GetOriNS(Fqdn(qname))
	if len(qns) > 0 {
		AddStrs(qns, resp)
		for _, v := range qns {
			GetNSs(v, resp)
		}
	} else {
		cns := GetOriCNAME(Fqdn(qname))
		if len(cns) > 0 {
			AddStr(cns, resp)
			GetNSs(cns, resp)
		}
	}
	return
}

func AddStr(qname string, resp *[]string) {
	for _, v := range *resp {
		if v == qname {
			return
		}
	}
	*resp = append(*resp, qname)
}

func AddStrs(qnames []string, resp *[]string) {
	for _, v := range qnames {
		ish := false
		for _, v2 := range *resp {
			if v2 == v {
				ish = true
				break
			}
		}
		if ish == false {

			*resp = append(*resp, v)
		}
	}

}

func recursion(socket *net.UDPConn, addr *net.UDPAddr, data []byte) {
	var req Msg
	req.Unpack(data)
	//var rn RequestNode
	if HasQuestion(req.Question[0].Name, req.Question[0].Qtype) {
		fmt.Println("already  ", req.Question[0].Name, req.Question[0].Qtype)
		return
	}

	rn := RNPool.Get().(RequestNode)
	rn.Req = &req
	rn.Conn = socket
	rn.Addr = addr
	rn.Qname = req.Question[0].Name
	rn.Qtype = req.Question[0].Qtype
	RNChan <- rn
	//Doit(rn, nil)
}

func Worker() {
	for {
		rn := <-RNChan
		go Doit(rn, nil)
	}
}

func ListenAndServe() {
	ua := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 53,
	}
	socket, err := net.ListenUDP("udp", &ua)
	//socket.w
	if err != nil {
		fmt.Println("监听失败", err)
	}

	defer socket.Close()

	for {
		data := make([]byte, 512)
		read, _, _, addr, err := socket.ReadMsgUDP(data, nil)

		if err != nil {
			fmt.Println("读取数据失败", err)
			return
		}
		go recursion(socket, addr, data[:read])
	}
}
