package planner

import (
	"encoding/gob"

	"github.com/lytics/dfa"

	"github.com/araddon/qlbridge/exec"
)

var (
	// States
	Starting    = dfa.State("starting")
	Running     = dfa.State("running")
	Resending   = dfa.State("resending")
	Finishing   = dfa.State("finishing")
	Exiting     = dfa.State("exiting")
	Terminating = dfa.State("terminating")
	// Letters
	Failure            = dfa.Letter("failure")
	SendFailure        = dfa.Letter("send-failure")
	SendSuccess        = dfa.Letter("send-success")
	FetchStateFailure  = dfa.Letter("fetch-state-failure")
	StoreStateFailure  = dfa.Letter("store-state-failure")
	EverybodyStarted   = dfa.Letter("everybody-started")
	EverybodyFinished  = dfa.Letter("everybody-finished")
	IndividualFinished = dfa.Letter("individual-finished")
	Exit               = dfa.Letter("exit")
)

func init() {
	gob.Register(ResultMsg{})
	gob.Register(DataMsg{})
}

type Conf struct {
	JobMaker    exec.JobMaker
	NodeCt      int
	GridName    string
	Hostname    string
	EtcdServers []string
	NatsServers []string
}

func (c *Conf) Clone() *Conf {
	if c.JobMaker == nil {
		panic("need job maker")
	}
	return &Conf{
		JobMaker:    c.JobMaker,
		NodeCt:      c.NodeCt,
		GridName:    c.GridName,
		Hostname:    c.Hostname,
		EtcdServers: c.EtcdServers,
		NatsServers: c.NatsServers,
	}
}

type DataMsg struct {
	Producer string
	Data     string
}

type ResultMsg struct {
	Producer string
	From     string
	Count    int
	Duration float64
}