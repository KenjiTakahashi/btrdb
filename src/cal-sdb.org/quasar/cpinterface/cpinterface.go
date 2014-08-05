package cpinterface

import (
	"cal-sdb.org/quasar"
	bstore "cal-sdb.org/quasar/bstoreEmu"
	"cal-sdb.org/quasar/qtree"
	capn "github.com/glycerine/go-capnproto"
	"log"
	"net"
	"sync"
)

func ServeCPNP(q *quasar.Quasar, ntype string, laddr string) {
	l, err := net.Listen(ntype, laddr)
	if err != nil {
		log.Panic(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}
		go func(c net.Conn) {
			DispatchCommands(q, c)
		}(conn)
	}
}

func DispatchCommands(q *quasar.Quasar, conn net.Conn) {
	//This governs the stream
	mtx := sync.Mutex{}
	for {
		mtx.Lock()
		seg, err := capn.ReadFromStream(conn, nil)
		if err != nil {
			log.Printf("ERR (%v) :: %v", conn.RemoteAddr(), err)
			conn.Close()
			break
		}
		mtx.Unlock()
		go func() {
			req := NewRootRequest(seg)
			rvseg := capn.NewBuffer(nil)
			resp := NewRootResponse(rvseg)
			resp.SetEchoTag(req.EchoTag())
			switch req.Which() {
			case REQUEST_QUERYSTANDARDVALUES:
				st := req.QueryStandardValues().StartTime()
				et := req.QueryStandardValues().EndTime()
				uuid := quasar.ConvertToUUID(req.QueryStandardValues().Uuid())
				ver := req.QueryStandardValues().Version()
				if ver == 0 {
					ver = bstore.LatestGeneration
				}
				rv, gen, err := q.QueryValues(uuid, st, et, ver)
				switch err {
				case nil:
					resp.SetStatusCode(STATUSCODE_OK)
					records := NewRecords(rvseg)
					rl := NewRecordList(rvseg, len(rv))
					rla := rl.ToArray()
					for i, v := range rv {
						rla[i].SetTime(v.Time)
						rla[i].SetValue(v.Val)
					}
					records.SetVersion(gen)
					records.SetValues(rl)
					resp.SetRecords(records)
				default:
					resp.SetStatusCode(STATUSCODE_INTERNALERROR)
					//TODO specialize this
				}
			case REQUEST_QUERYSTATISTICALVALUES:
				st := req.QueryStatisticalValues().StartTime()
				et := req.QueryStatisticalValues().EndTime()
				uuid := quasar.ConvertToUUID(req.QueryStatisticalValues().Uuid())
				pw := req.QueryStatisticalValues().PointWidth()
				ver := req.QueryStatisticalValues().Version()
				if ver == 0 {
					ver = bstore.LatestGeneration
				}
				rv, gen, err := q.QueryStatisticalValues(uuid, st, et, ver, pw)
				switch err {
				case nil:
					resp.SetStatusCode(STATUSCODE_OK)
					srecords := NewStatisticalRecords(rvseg)
					rl := NewStatisticalRecordList(rvseg, len(rv))
					rla := rl.ToArray()
					for i, v := range rv {
						rla[i].SetTime(v.Time)
						rla[i].SetCount(v.Count)
						rla[i].SetMin(v.Min)
						rla[i].SetMean(v.Mean)
						rla[i].SetMax(v.Max)
					}
					srecords.SetVersion(gen)
					srecords.SetValues(rl)
					resp.SetStatisticalRecords(srecords)
				default:
					resp.SetStatusCode(STATUSCODE_INTERNALERROR)
				}
				resp.SetStatusCode(STATUSCODE_INTERNALERROR)
			case REQUEST_QUERYVERSION:
				resp.SetStatusCode(STATUSCODE_INTERNALERROR)
			case REQUEST_QUERYNEARESTVALUE:
				resp.SetStatusCode(STATUSCODE_INTERNALERROR)
			case REQUEST_QUERYCHANGEDRANGES:
				resp.SetStatusCode(STATUSCODE_INTERNALERROR)
			case REQUEST_INSERTVALUES:
				uuid := quasar.ConvertToUUID(req.InsertValues().Uuid())
				rl := req.InsertValues().Values()
				rla := rl.ToArray()
				qtr := make([]qtree.Record, len(rla))
				for i, v := range rla {
					qtr[i] = qtree.Record{Time: v.Time(), Val: v.Value()}
				}
				q.InsertValues(uuid, qtr)
				//TODO add support for the sync variable
				resp.SetStatusCode(STATUSCODE_OK)
			case REQUEST_DELETEVALUES:
				resp.SetStatusCode(STATUSCODE_INTERNALERROR)
			}
			mtx.Lock()
			rvseg.WriteTo(conn)
			mtx.Unlock()
		}()
	}
}

/*
func EncodeMsg() *bytes.Buffer {
	rv := bytes.Buffer{}
	seg := capn.NewBuffer(nil)
	cmd := NewRootRequest(seg)
	ca := NewCmdA(seg)
	ca.SetCmd(50)
	cmd.SetCa(ca)
	seg.WriteTo(&rv)
	return &rv
}

func DecodeMsg(b *bytes.Buffer) {
	seg, err := capn.ReadFromStream(b, nil)
	if err != nil {
		log.Panic(err)
	}
	cmd := ReadRootRequest(seg)
	switch cmd.Which() {
	case REQUEST_CA:
		ca := cmd.Ca()
		log.Printf("ca val: %v", ca.Cmd())
	default:
		log.Printf("wtf")
	}
}
*/