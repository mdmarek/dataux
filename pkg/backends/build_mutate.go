package backends

import (
	u "github.com/araddon/gou"
	"github.com/araddon/qlbridge/exec"
	"github.com/araddon/qlbridge/expr"
)

var (
	_ = u.EMPTY
)

// func (m *Builder) VisitInsert(stmt *expr.SqlInsert) (interface{}, error) {
// 	u.Debugf("VisitInsert %+v", stmt)
// 	return nil, ErrNotImplemented
// }

func (m *Builder) VisitUpsert(stmt *expr.SqlUpsert) (interface{}, error) {
	u.Debugf("VisitUpsert %+v", stmt)
	return nil, ErrNotImplemented
}

func (m *Builder) VisitDelete(stmt *expr.SqlDelete) (interface{}, error) {
	u.Debugf("VisitDelete %+v", stmt)
	return nil, ErrNotImplemented
}

func (m *Builder) VisitUpdate(stmt *expr.SqlUpdate) (interface{}, error) {
	u.Debugf("VisitUpdate %+v", stmt)
	return nil, ErrNotImplemented
}

func (m *Builder) VisitInsert(stmt *expr.SqlInsert) (interface{}, error) {

	u.Debugf("VisitInsert %+v", stmt)
	tasks := make(exec.Tasks, 0)

	return tasks, nil
}