package models

import (
	"sync"
)

type TTLSets struct {
	_lock     *sync.RWMutex
	Asets     map[string]bool
	AAAAAsets map[string]bool
	MXsets    map[string]bool
	NSsets    map[string]bool
	CNsets    map[string]bool
}

type DataSets struct {
	_Alock    sync.RWMutex
	_AAAAlock sync.RWMutex
	_MXlock   sync.RWMutex
	_NSlock   sync.RWMutex
	_CNlock   sync.RWMutex

	Asets     map[string][]A
	AAAAAsets map[string][]AAAA
	MXsets    map[string]MX
	NSsets    map[string][]NS
	CNsets    map[string]CNAME
}

var (
	Dsmap  DataSets
	Ttlmap TTLSets
)

func InitTL() {
	Ttlmap._lock = new(sync.RWMutex)
	Ttlmap.Asets = make(map[string]bool)
	Ttlmap.AAAAAsets = make(map[string]bool)
	Ttlmap.MXsets = make(map[string]bool)
	Ttlmap.NSsets = make(map[string]bool)
	Ttlmap.CNsets = make(map[string]bool)
}

func TTLSet(key string, _type uint16) {
	Ttlmap._lock.Lock()
	defer Ttlmap._lock.Unlock()

	switch _type {
	case TypeA:
		Ttlmap.Asets[key] = true
	case TypeAAAA:
		Ttlmap.AAAAAsets[key] = true
	case TypeNS:
		Ttlmap.NSsets[key] = true
	case TypeCNAME:
		Ttlmap.CNsets[key] = true
	case TypeMX:
		Ttlmap.MXsets[key] = true
	}

}
func TTLExist(key string, _type uint16) bool {
	Ttlmap._lock.Lock()
	defer Ttlmap._lock.Unlock()
	ok := false
	switch _type {
	case TypeA:
		_, ok = Ttlmap.Asets[key]
	case TypeAAAA:
		_, ok = Ttlmap.AAAAAsets[key]
	case TypeNS:
		_, ok = Ttlmap.NSsets[key]
	case TypeCNAME:
		_, ok = Ttlmap.CNsets[key]
	case TypeMX:
		_, ok = Ttlmap.MXsets[key]
	}
	return ok
}
func TTLDel(key string, _type uint16) {
	Ttlmap._lock.Lock()
	defer Ttlmap._lock.Unlock()
	switch _type {
	case TypeA:
		delete(Ttlmap.Asets, key)
	case TypeAAAA:
		delete(Ttlmap.AAAAAsets, key)
	case TypeNS:
		delete(Ttlmap.NSsets, key)
	case TypeCNAME:
		delete(Ttlmap.CNsets, key)
	case TypeMX:
		delete(Ttlmap.MXsets, key)
	}
}

func InitDS() {
	//Dsmap._lock = new(sync.RWMutex)

	Dsmap.Asets = make(map[string][]A)
	Dsmap.AAAAAsets = make(map[string][]AAAA)
	Dsmap.NSsets = make(map[string][]NS)
	Dsmap.CNsets = make(map[string]CNAME)
	Dsmap.MXsets = make(map[string]MX)
}

func get_cnames(qname string) (cns []CNAME) {
	oldn := qname
	for uns := GetCNAME(oldn); len(uns.Target) > 0; uns = GetCNAME(oldn) {
		cns = append(cns, uns)
		oldn = uns.Target
	}
	return
}

func GetA(qname string) (addrs []A) {
	Dsmap._Alock.RLock()
	defer Dsmap._Alock.RUnlock()
	if v, ok := Dsmap.Asets[qname]; ok {
		addrs = v
	}
	return

}

func ExistA(qname string) (ish bool) {
	ish = false
	Dsmap._Alock.RLock()
	defer Dsmap._Alock.RUnlock()
	if _, ok := Dsmap.Asets[qname]; ok {
		ish = true
	}
	return
}

func GetAAAA(qname string) (addrs []AAAA) {
	Dsmap._AAAAlock.RLock()
	defer Dsmap._AAAAlock.RUnlock()
	if v, ok := Dsmap.AAAAAsets[qname]; ok {
		addrs = v
	}
	return
}

func GetNS(qname string) (strs []NS) {
	Dsmap._NSlock.RLock()
	defer Dsmap._NSlock.RUnlock()
	if v, ok := Dsmap.NSsets[qname]; ok {
		strs = v
	}
	return
}

func GetOriNS(qname string) (strs []string) {
	Dsmap._NSlock.RLock()
	defer Dsmap._NSlock.RUnlock()
	if v, ok := Dsmap.NSsets[qname]; ok {
		for _, v2 := range v {
			strs = append(strs, v2.Ns)
		}
	}
	return
}

func GetCNAME(qname string) (cn CNAME) {
	Dsmap._CNlock.RLock()
	defer Dsmap._CNlock.RUnlock()
	if v, ok := Dsmap.CNsets[qname]; ok {
		cn = v
	}
	return
}

func GetOriCNAME(qname string) (strs string) {
	Dsmap._CNlock.RLock()
	defer Dsmap._CNlock.RUnlock()
	if v, ok := Dsmap.CNsets[qname]; ok {
		strs = v.Target
	} else {
		strs = ""
	}
	return
}
func GetMX(qname string) (mrec MX) {
	Dsmap._MXlock.RLock()
	defer Dsmap._MXlock.RUnlock()
	if v, ok := Dsmap.MXsets[qname]; ok {
		mrec = v
	}
	return
}

func ResetA(name string) {
	Dsmap._Alock.Lock()
	defer Dsmap._Alock.Unlock()
	if _, ok := Dsmap.Asets[name]; !ok {
		Dsmap.Asets[name] = []A{}
	}
}

func AddA_str(name string, ip string) {
	//fmt.Println("AddA_str ", name, "  ", ip)
	var A1 A
	A1.Hdr.Class = ClassINET
	A1.Hdr.Rrtype = TypeA
	A1.Hdr.Rdlength = 4
	A1.Hdr.Name = name
	A1.A.UnmarshalText([]byte(ip))
	AddA(A1)
}

func AddA(arec A) {
	Dsmap._Alock.Lock()
	defer Dsmap._Alock.Unlock()
	ish := false
	for _, v := range Dsmap.Asets[arec.Header().Name] {
		if v.A.Equal(arec.A) {
			ish = true
			break
		}
	}
	if !ish {
		Dsmap.Asets[arec.Header().Name] = append(Dsmap.Asets[arec.Header().Name], arec)
	}
}

func AddAAAA_str(name string, ip string) {
	var A1 AAAA
	A1.Hdr.Class = ClassINET
	A1.Hdr.Rrtype = TypeA
	A1.Hdr.Rdlength = 16
	A1.Hdr.Name = name
	A1.AAAA.UnmarshalText([]byte(ip))
	AddAAAA(A1)
}

func AddAAAA(arec AAAA) {
	Dsmap._AAAAlock.Lock()
	defer Dsmap._AAAAlock.Unlock()
	ish := false
	for _, v := range Dsmap.AAAAAsets[arec.Header().Name] {
		if v.AAAA.Equal(arec.AAAA) {
			ish = true
			break
		}
	}
	if !ish {
		Dsmap.AAAAAsets[arec.Header().Name] = append(Dsmap.AAAAAsets[arec.Header().Name], arec)
	}
}

func AddNS(qname string, name string) {
	if qname == name {
		Log.Error("AddNS ererere %v %v", qname, name)
		panic("set ns")
	}

	Dsmap._NSlock.Lock()
	defer Dsmap._NSlock.Unlock()
	ish := false
	for _, v := range Dsmap.NSsets[qname] {
		if v.Ns == name {
			ish = true
			break
		}
	}
	if !ish {
		var ns NS
		ns.Hdr.Name = qname
		ns.Hdr.Rrtype = TypeNS
		ns.Hdr.Class = ClassINET
		ns.Hdr.Ttl = 86400
		ns.Ns = name
		Dsmap.NSsets[qname] = append(Dsmap.NSsets[qname], ns)
	}
}

func SetCNAME(qname string, names CNAME) {
	Dsmap._CNlock.Lock()
	defer Dsmap._CNlock.Unlock()
	Dsmap.CNsets[qname] = names
}

func SetOriCNAME(qname string, names string) {
	Dsmap._CNlock.Lock()
	defer Dsmap._CNlock.Unlock()
	var ns CNAME
	ns.Hdr.Name = qname
	ns.Hdr.Rrtype = TypeCNAME
	ns.Hdr.Class = ClassINET
	ns.Hdr.Ttl = 86400
	ns.Target = names

	Dsmap.CNsets[qname] = ns
}

func SetMX(qname string, mrec MX) {
	Dsmap._MXlock.Lock()
	defer Dsmap._MXlock.Unlock()
	Dsmap.MXsets[qname] = mrec
}

func SetA(qname string, arec []A) {
	Dsmap._Alock.Lock()
	defer Dsmap._Alock.Unlock()
	Dsmap.Asets[qname] = arec

}

func SetNS(qname string, names []NS) {
	for _, v := range names {
		if v.Ns == qname {
			Log.Error("ererere %v %v", qname, v)
			panic("set ns")
		}
	}
	Dsmap._NSlock.Lock()
	defer Dsmap._NSlock.Unlock()
	Dsmap.NSsets[qname] = names
}

func SetOriNS(qname string, names []string) {
	for _, v := range names {
		if v == qname {
			Log.Error("SetOriNS %v %v", qname, v)
			panic("SetOriNS set ns")
		}
	}
	Dsmap._NSlock.Lock()
	defer Dsmap._NSlock.Unlock()
	Dsmap.NSsets[qname] = []NS{}
	for _, v := range names {
		var ns NS
		ns.Hdr.Name = qname
		ns.Hdr.Rrtype = TypeNS
		ns.Hdr.Class = ClassINET
		ns.Hdr.Ttl = 86400
		ns.Ns = v
		Dsmap.NSsets[qname] = append(Dsmap.NSsets[qname], ns)
	}

}
