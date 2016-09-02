package models

import (
	"fmt"
	"sync/atomic"
	//"fmt"
	"net"
	"strconv"
	"sync"
	"time"
	//"time"
)

type ServerUnit struct {
	Answer      []RR
	AnswerCname []CNAME
	NSinfo      []NS
	NSA         []A
	NSCname     []CNAME
	isout       bool
	issoa       bool
}

type au_req struct {
	flag uint32
	wc   chan ServerUnit
}

func (this *ServerUnit) fromMsg(packet Msg) {
	for _, v := range packet.Answer {

		switch v.(type) {
		case *A:
			this.Answer = append(this.Answer, v)
		case *AAAA:
			this.Answer = append(this.Answer, v)
		case *MX:
			this.Answer = append(this.Answer, v)
		case *CNAME:
			this.AnswerCname = append(this.AnswerCname, *(v.(*CNAME)))
		case *NS:
			this.NSinfo = append(this.NSinfo, *(v.(*NS)))
		}
	}

	for _, v := range packet.Ns {
		switch v.(type) {
		case *A:
			this.NSA = append(this.NSA, *(v.(*A)))
		case *CNAME:
			this.NSCname = append(this.NSCname, *(v.(*CNAME)))
		case *NS:
			this.NSinfo = append(this.NSinfo, *(v.(*NS)))
		}
	}

	for _, v := range packet.Extra {
		switch v.(type) {
		case *A:
			this.NSA = append(this.NSA, *(v.(*A)))
		case *CNAME:
			this.NSCname = append(this.NSCname, *(v.(*CNAME)))
		case *NS:
			this.NSinfo = append(this.NSinfo, *(v.(*NS)))
		}
	}

}

type ServerMux struct {
	ServNums int
	_lock    *sync.RWMutex
	MuxMap   map[string]([]au_req)
	cons     []*net.UDPConn
}

func (this *ServerMux) InitServer(nums int) {
	this._lock = new(sync.RWMutex)
	this.MuxMap = make(map[string][]au_req)
	this.ServNums = nums
	for i := 0; i < this.ServNums; i++ {
		ua := net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: i + 3000,
		}

		socket, err := net.ListenUDP("udp", &ua)
		if err != nil {
			checkErr(err)
			return
		}
		this.cons = append(this.cons, socket)
		go this.serve(i)
	}

}

func addNs(nmap map[string][]string, qname string, name string) {
	ish := false
	for _, v := range nmap[qname] {
		if v == name {
			ish = true
			break
		}
	}
	if !ish {
		nmap[qname] = append(nmap[qname], name)
	}
}

func addA(nmap map[string][]A, arec A) {
	ish := false

	for _, v := range nmap[arec.Header().Name] {
		if v.A.Equal(arec.A) {
			ish = true
			break
		}
	}
	if !ish {
		nmap[arec.Header().Name] = append(nmap[arec.Header().Name], arec)
	}
}

func Serves() {
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
		}
		go recursion(socket, addr, data[:read])
	}
}

func (this *ServerMux) serve(idx int) {

	socket := this.cons[idx]

	defer socket.Close()

	for {
		data := make([]byte, 512)
		//read, _, err := socket.ReadFrom(data)
		read, _, _, _, err := socket.ReadMsgUDP(data, nil)
		if err != nil {
			checkErr(err)
			continue
		}

		var resp Msg
		var ds DataSet
		resp.Unpack(data[:read])
		ds.fromMsg(resp)
		//fmt.Printf("ds %+v\n", ds)
		issoa := false

		if len(resp.Answer) == 0 && len(resp.Ns) == 0 && len(resp.Extra) == 0 {
			issoa = true
		} else {
			//fmt.Printf("resp ANSWER %+v\n", resp)
		}

		if !issoa {
			for _, v := range resp.Answer {
				switch v.(type) {
				case *SOA:
					issoa = true
				}
			}
		}

		if !issoa {
			for _, v := range resp.Ns {
				switch v.(type) {
				case *SOA:
					issoa = true
				}
			}
		}

		if !issoa {
			for _, v := range resp.Extra {
				switch v.(type) {
				case *SOA:
					issoa = true
				}
			}
		}

		if issoa == false {
			localA := make(map[string][]A)
			localNS := make(map[string][]string)
			localCN := make(map[string]string)
			//var nsttl uint32
			//var cnttl uint32
			for _, v := range resp.Answer {
				switch v.(type) {
				case *NS:
					//nsttl = v.Header().Ttl
					if v.(*NS).Header().Name == v.(*NS).Ns {
						Log.Error("v.(*NS).Header( %v %v", v.(*NS).Header(), v.(*NS).Ns)
						panic("add ns")
					}
					addNs(localNS, v.(*NS).Header().Name, v.(*NS).Ns)
					if _, ok := localA[v.(*NS).Ns]; !ok {
						localA[v.(*NS).Ns] = []A{}
					}
					//AddNS(v.(*NS).Header().Name, v.(*NS).Ns)
					//ResetA(v.(*NS).Ns)
				}
			}
			for _, v := range resp.Ns {
				switch v.(type) {
				case *NS:
					//nsttl = v.Header().Ttl
					if v.(*NS).Header().Name == v.(*NS).Ns {
						Log.Error("v.(*NS).Header( %+v %v", resp, v.(*NS).Ns)
						panic("add ns")
					}
					addNs(localNS, v.(*NS).Header().Name, v.(*NS).Ns)
					if _, ok := localA[v.(*NS).Ns]; !ok {
						localA[v.(*NS).Ns] = []A{}
					}
					//AddNS(v.(*NS).Header().Name, v.(*NS).Ns)
					//	ResetA(v.(*NS).Ns)
				}
			}
			for _, v := range resp.Extra {
				switch v.(type) {
				case *NS:
					//nsttl = v.Header().Ttl
					if v.(*NS).Header().Name == v.(*NS).Ns {
						Log.Error("v.(*NS).Header( %v %v", v.(*NS).Header(), v.(*NS).Ns)
						panic("add ns")
					}
					
					addNs(localNS, v.(*NS).Header().Name, v.(*NS).Ns)
					if _, ok := localA[v.(*NS).Ns]; !ok {
						localA[v.(*NS).Ns] = []A{}
					}
					//AddNS(v.(*NS).Header().Name, v.(*NS).Ns)
					//ResetA(v.(*NS).Ns)
				}
			}

			/////////////////////////////////////////////////////////////////////////////////////

			for _, v := range resp.Answer {
				switch v.(type) {
				case *CNAME:
					ish := ExistA(v.(*CNAME).Header().Name)
					if !ish {
						_, ish = localA[v.(*CNAME).Header().Name]
					}
					if ish {
						//cnttl = v.Header().Ttl
						localCN[v.(*CNAME).Header().Name] = v.(*CNAME).Target
						//ish2 := ExistA(v.(*CNAME).Target)
						//if !ish2 {
						//ResetA(v.(*CNAME).Header().Name)
						//}
						//SetCNAME(v.(*CNAME).Header().Name, v.(*CNAME).Target)
					}
				}
			}
			for _, v := range resp.Ns {
				switch v.(type) {
				case *CNAME:
					ish := ExistA(v.(*CNAME).Header().Name)
					if !ish {
						_, ish = localA[v.(*CNAME).Header().Name]
					}
					if ish {
						//cnttl = v.Header().Ttl
						localCN[v.(*CNAME).Header().Name] = v.(*CNAME).Target
						//ish2 := ExistA(v.(*CNAME).Target)
						//if !ish2 {
						//ResetA(v.(*CNAME).Header().Name)
						//}
						//SetCNAME(v.(*CNAME).Header().Name, v.(*CNAME).Target)
					}
				}
			}
			for _, v := range resp.Extra {
				switch v.(type) {
				case *CNAME:
					ish := ExistA(v.(*CNAME).Header().Name)
					if !ish {
						_, ish = localA[v.(*CNAME).Header().Name]
					}
					if ish {
						//	cnttl = v.Header().Ttl
						localCN[v.(*CNAME).Header().Name] = v.(*CNAME).Target
						//ish2 := ExistA(v.(*CNAME).Target)
						//if !ish2 {
						//ResetA(v.(*CNAME).Header().Name)
						//}
						//SetCNAME(v.(*CNAME).Header().Name, v.(*CNAME).Target)
					}
				}
			}

			////////////////////////////////////////////////////////////////////////////////

			for _, v := range resp.Answer {
				switch v.(type) {
				case *A:
					ish := ExistA(v.(*A).Header().Name)
					if !ish {
						_, ish = localA[v.(*A).Header().Name]
					}
					if ish {
						addA(localA, *v.(*A))
						//AddA(*v.(*A))
					}

				}
			}
			for _, v := range resp.Ns {
				switch v.(type) {
				case *A:
					ish := ExistA(v.(*A).Header().Name)
					if !ish {
						_, ish = localA[v.(*A).Header().Name]
					}
					if ish {
						addA(localA, *v.(*A))
						//AddA(*v.(*A))
					}

				}
			}
			for _, v := range resp.Extra {
				switch v.(type) {
				case *A:
					ish := ExistA(v.(*A).Header().Name)
					if !ish {
						_, ish = localA[v.(*A).Header().Name]
					}
					if ish {
						addA(localA, *v.(*A))
						//AddA(*v.(*A))
					}

				}
			}

			for k, v := range localA {
				SetA(k, v)
				//if !TTLExist(k, TypeA) {
				//TTLSet(k, TypeA)
				//AddTimer(k, TypeA, ttlcb, 12)
				//}

			}
			for k, v := range localNS {
				SetOriNS(k, v)
				//if nsttl > 0 {
				//if !TTLExist(k, TypeNS) {
				//TTLSet(k, TypeNS)
				//AddTimer(k, TypeNS, ttlcb, 13)
				//}
				//}

			}

			for k, v := range localCN {
				SetOriCNAME(k, v)
				//if cnttl > 0 {
				//if !TTLExist(k, TypeCNAME) {
				//	TTLSet(k, TypeCNAME)
				//	AddTimer(k, TypeCNAME, ttlcb, 14)
				//}
				//}

			}

			var rns []RequestNode
			func() {
				ReqsLock.Lock()
				defer ReqsLock.Unlock()
				switch resp.Question[0].Qtype {
				case TypeA:
					if v, ok := ReqA[resp.Question[0].Name]; ok {
						rns = v
						Log.Info("delete %v %v", resp.Question[0].Name, len(v))
						delete(ReqA, resp.Question[0].Name)
					}
				case TypeAAAA:
					if v, ok := ReqAAAA[resp.Question[0].Name]; ok {
						rns = v
						Log.Info("delete %v %v", resp.Question[0].Name, len(v))
						delete(ReqAAAA, resp.Question[0].Name)
					}
				case TypeNS:
					if v, ok := ReqNS[resp.Question[0].Name]; ok {
						rns = v
						Log.Info("delete %v %v", resp.Question[0].Name, len(v))
						delete(ReqNS, resp.Question[0].Name)
					}
				case TypeMX:
					if v, ok := ReqMX[resp.Question[0].Name]; ok {
						rns = v
						Log.Info("delete %v %v", resp.Question[0].Name, len(v))
						delete(ReqMX, resp.Question[0].Name)
					}
				}
			}()
			for _, v := range rns {
				go Doit(v, &ds)
			}

		}

	}
}

func (this *ServerMux) Request(qname string, qtype uint16, addrs []net.IP) (su ServerUnit) {
	idx := strconv.Itoa(int(qtype)) + qname
	var req Msg
	//fmt.Println("Request  wait")
	req.SetQuestion(qname, qtype)
	//wc := make(chan ServerUnit)
	var wc au_req
	wc.wc = make(chan ServerUnit)
	func() {
		this._lock.Lock()
		defer this._lock.Unlock()
		this.MuxMap[idx] = append(this.MuxMap[idx], wc)
	}()
	socket := this.cons[int(Id())%this.ServNums]
	buf, _ := req.PackBuffer(nil)
	for _, v := range addrs {
		//fmt.Println("req for ip: ", v.String())
		//strip, _ := v.MarshalText()

		//adr, _ := net.ResolveUDPAddr("udp", string(strip)+":"+strconv.Itoa(53))
		var adr net.UDPAddr
		adr.IP = v
		adr.Port = 53
		//socket.WriteTo(buf, adr)
		socket.WriteMsgUDP(buf, nil, &adr)
	}

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	Log.Debug("req  wait")
	select {
	case su = <-wc.wc:
	case <-tick.C:
		if atomic.CompareAndSwapUint32(&wc.flag, 0, 1) {
			close(wc.wc)
		}
		su.isout = true
		Log.Debug("Request case <-tick.C")
	}
	//fmt.Println("close(wc) ")

	//tick.Stop()

	return su
}

func (this *ServerMux) Request2(qname string, qtype uint16, addrs []net.IP) {
	//fmt.Println("Request2 ", qname, qtype)
	var req Msg
	req.SetQuestion(qname, qtype)
	socket := this.cons[int(Id())%this.ServNums]
	buf, _ := req.PackBuffer(nil)
	for _, v := range addrs {
		//fmt.Println("req for ip: ", v.String())
		//strip, _ := v.MarshalText()

		//adr, _ := net.ResolveUDPAddr("udp", string(strip)+":"+strconv.Itoa(53))
		var adr net.UDPAddr
		adr.IP = v
		adr.Port = 53
		//socket.WriteTo(buf, adr)
		socket.WriteMsgUDP(buf, nil, &adr)
	}
}
