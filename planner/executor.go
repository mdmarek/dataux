package planner

import (
	"fmt"

	u "github.com/araddon/gou"
	"github.com/lytics/grid"

	"github.com/araddon/qlbridge/exec"
	"github.com/araddon/qlbridge/plan"

	"github.com/dataux/dataux/planner/gridrunner"
)

var (
	// ensure it meets interfaces
	_ exec.Executor = (*ExecutorGrid)(nil)

	_ = u.EMPTY
)

// Build a Sql Job which may be a Grid/Distributed job
func BuildSqlJob(ctx *plan.Context, gs *gridrunner.Server) (*ExecutorGrid, error) {
	sqlPlanner := plan.NewPlanner(ctx)
	baseJob := exec.NewExecutor(ctx, sqlPlanner)

	job := &ExecutorGrid{JobExecutor: baseJob}
	job.Executor = job
	job.GridServer = gs
	job.Ctx = ctx
	task, err := exec.BuildSqlJobPlanned(job.Planner, job.Executor, ctx)
	if err != nil {
		return nil, err
	}
	taskRunner, ok := task.(exec.TaskRunner)
	if !ok {
		return nil, fmt.Errorf("Expected TaskRunner but was %T", task)
	}
	job.RootTask = taskRunner
	return job, err
}

// Sql job that wraps the generic qlbridge job builder
// - contains ref to the shared GridServer which has info to
//   distribute tasks across servers
type ExecutorGrid struct {
	*exec.JobExecutor
	GridServer *gridrunner.Server
}

// Finalize is after the Dag of Relational-algebra tasks have been assembled
//  and just before we run them.
func (m *ExecutorGrid) Finalize(resultWriter exec.Task) error {
	u.Debugf("planner.Finalize  %#v", m.JobExecutor.RootTask)

	m.JobExecutor.RootTask.Add(resultWriter)
	m.JobExecutor.Setup()
	u.Debugf("finished finalize")
	return nil
}

func (m *ExecutorGrid) WalkSelect(p *plan.Select) (exec.Task, error) {
	if len(p.Stmt.From) > 0 {
		u.Debugf("ExecutorGrid.WalkSelect ?  %s", p.Stmt.Raw)
	}

	if len(p.Stmt.With) > 0 && p.Stmt.With.Bool("distributed") {
		u.Warnf("has distributed!!!!!: %#v", p.Stmt.With)

		sqlTask, err := m.JobExecutor.WalkSelect(p)
		if err != nil {
			u.Errorf("Could not create select task %v", err)
			return nil, err
		}

		localTask := exec.NewTaskSequential(m.Ctx)
		taskUint, err := gridrunner.NextId()
		if err != nil {
			u.Errorf("Could not create task id %v", err)
			return nil, err
		}
		flow := gridrunner.NewFlow(taskUint)

		rx, err := grid.NewReceiver(m.GridServer.Grid.Nats(), flow.Name(), 2, 0)
		if err != nil {
			u.Errorf("%v: error: %v", "ourtask", err)
			return nil, err
		}
		natsSource := NewSourceNats(m.Ctx, rx)
		localTask.Add(natsSource)

		tx, err := grid.NewSender(m.GridServer.Grid.Nats(), 1)
		if err != nil {
			u.Errorf("error: %v", err)
		}
		natsSink := NewSinkNats(m.Ctx, flow.Name(), tx)
		sqlTask.Add(natsSink)

		// submit task in background node
		go func() {
			m.GridServer.SubmitTask(localTask, flow, sqlTask, p) // task submission to worker actors
			// need to send signal to quit
			ch := natsSource.MessageOut()
			close(ch)
		}()
		return localTask, nil
	}

	return m.JobExecutor.WalkSelect(p)
}