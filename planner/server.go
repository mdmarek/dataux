package planner

import (
	"fmt"
	"os"
	"time"

	u "github.com/araddon/gou"
	"github.com/lytics/grid"
	"github.com/lytics/grid/condition"
	"github.com/sony/sonyflake"

	"github.com/araddon/qlbridge/datasource"
	"github.com/araddon/qlbridge/exec"
	"github.com/araddon/qlbridge/plan"
)

var sf *sonyflake.Sonyflake

func init() {
	var st sonyflake.Settings
	// TODO, ensure we get a unique etcdid for machineid
	st.StartTime = time.Now()
	sf = sonyflake.NewSonyflake(st)
	// Lets use our distributed generator
	plan.NextId = NextIdUnsafe
}

func NextIdUnsafe() uint64 {
	uv, err := NextId()
	if err != nil {
		u.Errorf("error generating nextId %v", err)
	}
	return uv
}

func NextId() (uint64, error) {
	return sf.NextID()
}

// Server that manages the sql tasks, workers
type Server struct {
	Conf       *Conf
	reg        *datasource.Registry
	Grid       grid.Grid
	started    bool
	lastTaskId uint64
	quit       chan bool
	exit       <-chan bool
}

func (s *Server) Init(actorMaker grid.ActorMaker) error {

	u.Debugf("%p grid etcd: %v nats:%v", s, s.Conf.EtcdServers, s.Conf.NatsServers)

	if s.Grid != nil {
		u.Warnf("ack, replacing grid?")
	}
	// We are going to start a "Grid" with specified maker
	//   - nilMaker = "master" only used for submitting tasks, not performing them
	//   - normal maker;  performs specified work units
	s.Grid = grid.New(s.Conf.GridName, s.Conf.Hostname, s.Conf.EtcdServers, s.Conf.NatsServers, actorMaker)
	exit, err := s.Grid.Start()
	if err != nil {
		u.Errorf("failed to start grid: %v", err)
		return fmt.Errorf("error starting grid %v", err)
	}
	s.exit = exit
	return nil
}
func (s *Server) InitMaster() error {
	return s.Init(&nilMaker{})
}
func (s *Server) InitWorker() error {
	u.Debugf("%p starting grid worker", s)
	actor, err := newActorMaker(s.Conf)
	if err != nil {
		u.Errorf("failed to make actor maker: %v", err)
		return err
	}
	return s.Init(actor)
}

// Submits a Sql Select statement task for planning across multiple nodes
func (s *Server) RunSqlMaster(completionTask exec.TaskRunner, ns *SourceNats, flow Flow, p *plan.Select) error {

	t := newSqlMasterTask(s, completionTask, ns, flow, p)
	return t.Run()
}

func (s *Server) Run(quit chan bool) error {
	return s.runMaker(quit)
}

func (s *Server) runMaker(quit chan bool) error {

	defer func() {
		u.Debugf("defer grid worker complete: %s", s.Conf.Hostname)
		s.Grid.Stop()
	}()

	s.quit = quit

	complete := make(chan bool)

	j := condition.NewJoin(s.Grid.Etcd(), 30*time.Second, s.Grid.Name(), "hosts", s.Conf.Hostname)
	u.Infof("Grid:%p join %v", s.Grid.Etcd(), s.Grid.Name())
	err := j.Join()
	if err != nil {
		u.Errorf("failed to register grid node: %v", err)
		os.Exit(1)
	}
	defer j.Exit()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.quit:
				u.Debugf("quit signal")
				close(complete)
				return
			case <-s.exit:
				u.Debugf("worker grid exit??")
				return
			case <-ticker.C:
				err := j.Alive()
				if err != nil {
					u.Errorf("failed to report liveness: %v", err)
					os.Exit(1)
				}
			}
		}
	}()

	w := condition.NewCountWatch(s.Grid.Etcd(), s.Grid.Name(), "hosts")
	defer w.Stop()

	waitForCt := s.Conf.NodeCt + 1 // worker nodes + master
	u.Debugf("%p %s  waiting for %d nodes to join", s, s.Grid.Name(), waitForCt)
	//u.LogTraceDf(u.WARN, 16, "")
	started := w.WatchUntil(waitForCt)
	u.Infof("found the watch!!!!")
	select {
	case <-complete:
		u.Debugf("got complete signal")
		return nil
	case <-s.exit:
		u.Debug("Shutting down, grid exited")
		return nil
	case <-w.WatchError():
		u.Errorf("failed to watch other hosts join: %v", err)
		os.Exit(1)
	case <-started:
		s.started = true
		//u.Debugf("%p now started", s)
	}
	<-s.exit
	//u.Debug("shutdown complete")
	return nil
}

type Flow string

func NewFlow(nr uint64) Flow {
	return Flow(fmt.Sprintf("sql-%v", nr))
}

func (f Flow) NewContextualName(name string) string {
	return fmt.Sprintf("%v-%v", f, name)
}

func (f Flow) Name() string {
	return string(f)
}

func (f Flow) String() string {
	return string(f)
}
