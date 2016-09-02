package models

import (
	"time"
	//"fmt"
	"sync"
	//"fmt"
	"net"
	//"runtime/pprof"
	//"strconv"
	"strings"
	//"sync"
	//"sync/atomic"
	//"time"
)

type RequestNode struct {
	Qname    string
	Qtype    uint16
	LastName string
	LastType uint16
	Retrys   int
	TPoint   int64
	Cns      []CNAME
	Req      *Msg
	Conn     *net.UDPConn
	Addr     *net.UDPAddr
}

type DataSet struct {
	arecs   []A
	a64recs []AAAA
	mxrecs  []MX
	cnames  []CNAME
	nss     []NS
}

func (this *DataSet) DataSet_Cname(qname string) (cn []CNAME) {
	for _, v := range this.cnames {
		if v.Header().Name == qname {
			cn = append(cn, v)
			cn = append(cn, this.DataSet_Cname(v.Target)...)

			break
		}
	}
	return
}

func (this *DataSet) DataSet_A(qname string) (res []A, ok bool) {
	for _, v := range this.arecs {
		if v.Header().Name == qname {
			res = append(res, v)
			ok = true
		}
	}
	return
}

func (this *DataSet) DataSet_AAAA(qname string) (res []AAAA, ok bool) {
	for _, v := range this.a64recs {
		if v.Header().Name == qname {
			res = append(res, v)
			ok = true
		}
	}
	return
}

func (this *DataSet) DataSet_NS(qname string) (res []NS, ok bool) {
	for _, v := range this.nss {
		if v.Header().Name == qname {
			res = append(res, v)
			ok = true
		}
	}
	return
}

func (this *DataSet) DataSet_MX(qname string) (res MX, ok bool) {
	for _, v := range this.mxrecs {
		if v.Header().Name == qname {
			res = v
			ok = true
		}
	}
	return
}

func (this *DataSet) fromMsg(resp Msg) {
	for _, v := range resp.Answer {
		switch v.(type) {
		case *A:
			this.arecs = append(this.arecs, *v.(*A))
		case *AAAA:
			this.a64recs = append(this.a64recs, *v.(*AAAA))
		case *CNAME:
			this.cnames = append(this.cnames, *v.(*CNAME))
		case *NS:
			this.nss = append(this.nss, *v.(*NS))
		case *MX:
			this.mxrecs = append(this.mxrecs, *v.(*MX))
		}
	}

	for _, v := range resp.Ns {
		switch v.(type) {
		case *A:
			this.arecs = append(this.arecs, *v.(*A))
		case *AAAA:
			this.a64recs = append(this.a64recs, *v.(*AAAA))
		case *CNAME:
			this.cnames = append(this.cnames, *v.(*CNAME))
		case *NS:
			this.nss = append(this.nss, *v.(*NS))
		case *MX:
			this.mxrecs = append(this.mxrecs, *v.(*MX))
		}
	}

	for _, v := range resp.Extra {
		switch v.(type) {
		case *A:
			this.arecs = append(this.arecs, *v.(*A))
		case *AAAA:
			this.a64recs = append(this.a64recs, *v.(*AAAA))
		case *CNAME:
			this.cnames = append(this.cnames, *v.(*CNAME))
		case *NS:
			this.nss = append(this.nss, *v.(*NS))
		case *MX:
			this.mxrecs = append(this.mxrecs, *v.(*MX))
		}
	}
	return
}

var (
	ReqsLock sync.RWMutex
	ReqA     map[string][]RequestNode
	ReqNS    map[string][]RequestNode
	ReqAAAA  map[string][]RequestNode
	ReqMX    map[string][]RequestNode
)

func AddQuestion(rn RequestNode) {
	//Log.Info("add q %+v", rn)
	rn.TPoint = time.Now().Unix()
	ReqsLock.Lock()
	defer ReqsLock.Unlock()
	switch rn.Qtype {
	case TypeA:
		ReqA[rn.Qname] = append(ReqA[rn.Qname], rn)
	case TypeNS:
		ReqNS[rn.Qname] = append(ReqNS[rn.Qname], rn)
	case TypeAAAA:
		ReqAAAA[rn.Qname] = append(ReqAAAA[rn.Qname], rn)
	case TypeMX:
		ReqMX[rn.Qname] = append(ReqMX[rn.Qname], rn)
	}
}

func HasQuestion(qname string, qtype uint16) (ok bool) {

	ReqsLock.RLock()
	defer ReqsLock.RUnlock()

	switch qtype {
	case TypeA:
		_, ok = ReqA[qname]

	case TypeNS:
		_, ok = ReqNS[qname]

	case TypeAAAA:
		_, ok = ReqAAAA[qname]

	case TypeMX:
		_, ok = ReqMX[qname]

	}
	return
}

func Over(req RequestNode, res []RR) {
	if req.Req != nil && req.Addr != nil && req.Conn != nil {
		var resp Msg
		resp.SetReply(req.Req)

		if len(req.Cns) > 0 {

			for _, av := range req.Cns {
				av2 := av
				resp.Answer = append(resp.Answer, &av2)
			}
		}
		resp.Answer = append(resp.Answer, res...)
		msg, _ := resp.PackBuffer(nil)
		req.Conn.WriteMsgUDP(msg, nil, req.Addr)

	} else if len(req.LastName) > 0 {
		//Log.Info("Over req %+v", req)
		for _, v := range res {
			switch v.Header().Rrtype {
			case TypeA:
				var reqMsg Msg
				//fmt.Println("Request  wait")
				reqMsg.SetQuestion(req.LastName, req.LastType)
				//Log.Info("reqMsg.SetQuestion %+v", v.(*A))
				//socket := this.cons[int(Id())%this.ServNums]
				buf, _ := reqMsg.PackBuffer(nil)

				var adr net.UDPAddr
				adr.IP = v.(*A).A
				adr.Port = 53
				req.Conn.WriteMsgUDP(buf, nil, &adr)

			}
		}

	}
	RNPool.Put(req)
}

func Doit(req RequestNode, ds *DataSet) {
	req.Retrys++
	if req.Retrys > 30 {
		RNPool.Put(req)
		return
	}
	//Log.Info("doit %v", req)
	qname := req.Qname
	qtype := req.Qtype
	var qncms []CNAME
	if ds != nil {
		qncms = ds.DataSet_Cname(qname)
		req.Cns = append(req.Cns, qncms...)

	}

	if len(qncms) > 0 {
		qname = qncms[len(qncms)-1].Target
		req.Qname = qname
	}

	switch qtype {
	case TypeA:
		var ans []A
		if ds != nil {
			ans, _ = ds.DataSet_A(qname)
		}
		if len(ans) == 0 {
			ans = GetA(qname)
		}
		if len(ans) > 0 {
			//var ti interface{}
			//ti = ans
			var rrs []RR
			for _, av := range ans {
				av2 := av
				rrs = append(rrs, &av2)
			}

			Over(req, rrs)
			return
		}

	case TypeAAAA:
		var ans []AAAA
		if ds != nil {
			ans, _ = ds.DataSet_AAAA(qname)
		}
		if len(ans) == 0 {
			ans = GetAAAA(qname)
		}
		if len(ans) > 0 {
			var rrs []RR
			for _, av := range ans {
				av2 := av
				rrs = append(rrs, &av2)
			}
			Over(req, rrs)
			return
		}
	case TypeNS:
		var ans []NS
		if ds != nil {
			ans, _ = ds.DataSet_NS(qname)
		}
		if len(ans) == 0 {
			ans = GetNS(qname)
		}
		if len(ans) > 0 {
			var rrs []RR
			for _, av := range ans {
				av2 := av
				rrs = append(rrs, &av2)
			}
			Over(req, rrs)
			return
		}
	case TypeMX:
		var ans MX
		if ds != nil {
			ans, _ = ds.DataSet_MX(qname)
		}
		if len(ans.Mx) == 0 {
			ans = GetMX(qname)
		}
		if len(ans.Mx) > 0 {
			var anss []MX
			anss = append(anss, ans)
			var rrs []RR
			for _, av := range anss {
				av2 := av
				rrs = append(rrs, &av2)
			}
			Over(req, rrs)
			return
		}
	}

	ns := []string{}
	if ds != nil {
		if v, ok := ds.DataSet_NS(qname); ok {

			for _, v2 := range v {
				if v2.Ns == req.Qname {
					Log.Error("v2.Ns == req.Qname %v %v", qname, v)
				} else {
					ns = append(ns, v2.Ns)
				}

			}
		}
	}
	//isov := false
	if len(ns) == 0 {
		ns = GetOriNS(qname)
		for _, nv := range ns {
			if nv == req.Qname {
				ns = []string{}
				break
			}
		}
	}

	for len(ns) == 0 {
		if qname == "." {
			Log.Error(". ns not found ! ")
			return
		}
		// 剥
		labs := strings.SplitN(qname, ".", 2)
		if len(labs[1]) == 0 {
			qname = "."
		} else {
			qname = labs[1]
		}

		ns = GetOriNS(qname)
		for _, nv := range ns {
			if nv == req.Qname {
				ns = []string{}
				break
			}
		}
	}

	if len(ns) == 0 {
		Log.Error("ns not found %v %v", qname, qtype)
		return
	}
	var nips []net.IP
	for _, v := range ns {
		rn := v
		qnms := get_cnames(v)
		if len(qnms) > 0 {
			rn = qnms[len(qnms)-1].Target
		}
		ish := false
		if ds != nil {
			if v2, ok := ds.DataSet_A(rn); ok {
				for _, av := range v2 {
					nips = append(nips, av.A)
				}
				ish = true
			}
		}
		if ish == false {
			nsa := GetA(rn)
			for _, av := range nsa {
				nips = append(nips, av.A)
			}
		}
	}
	//req.Qname = qname
	AddQuestion(req)
	if len(nips) > 0 {
		Server.Request2(req.Qname, qtype, nips)
	} else {
		for _, v := range ns {
			//var rn RequestNode
			rn := RNPool.Get().(RequestNode)
			rn.Qtype = TypeA
			rn.Qname = v
			rn.LastName = req.Qname
			rn.LastType = qtype
			rn.Conn = Server.cons[Id()%uint16(Server.ServNums)]
			AddQuestion(rn)
			go Doit(rn, nil)

		}
	}
}
