package frontends

import (
	"database/sql/driver"
	//"io"

	u "github.com/araddon/gou"
	"github.com/araddon/qlbridge/datasource"
	"github.com/araddon/qlbridge/exec"
	"github.com/araddon/qlbridge/expr"
	"github.com/araddon/qlbridge/value"
	"github.com/dataux/dataux/pkg/models"
	"github.com/dataux/dataux/vendor/mixer/mysql"
)

var (
	// What is the interface we want to implement here?
	_ exec.TaskRunner = (*MysqlResultWriter)(nil)
)

type MysqlResultWriter struct {
	//resp         models.ResultProvider
	//cols         []*models.ResultColumn
	writer       models.ResultWriter
	schema       *models.Schema
	Rs           *mysql.Resultset
	projection   *expr.Projection
	wroteHeaders bool
	*exec.TaskBase
}

func NewMysqlResultWriter(writer models.ResultWriter, proj *expr.Projection, schema *models.Schema) *MysqlResultWriter {

	m := &MysqlResultWriter{writer: writer, projection: proj, schema: schema}
	m.TaskBase = exec.NewTaskBase("MysqlResultWriter")
	m.Rs = mysql.NewResultSet()
	m.Handler = resultWrite(m)
	return m
}
func (m *MysqlResultWriter) Close() error {
	u.Infof("in mysql resultwriter Close()")
	m.writer.WriteResult(m.Rs)
	return nil
}

func resultWrite(m *MysqlResultWriter) exec.MessageHandler {

	return func(_ *exec.Context, msg datasource.Message) bool {

		u.Debugf("in resultWrite:  %#v", msg)
		if !m.wroteHeaders {
			m.WriteHeaders()
		}

		select {
		case <-m.SigChan():
			return false
		default:
			//ok
		}

		if vals, ok := msg.Body().([]driver.Value); ok {
			u.Debugf("got msg in result writer: %#v", vals)
			m.Rs.AddRowValues(vals)
			return true
		}

		u.Errorf("could not convert to message reader: %T", msg.Body())
		return false
	}
}

func (m *MysqlResultWriter) WriteHeaders() error {

	cols := m.projection.Columns
	u.Debugf("writing mysql headers: %v", cols)
	if len(cols) == 0 {
		u.Warnf("Wat?   no columns?   %v", 0)
		return nil
	}
	m.wroteHeaders = true
	s := m.schema
	if s == nil {
		panic("no schema")
	}
	for i, col := range cols {
		as := col.Name
		if col.Col != nil {
			as = col.Col.As
		}
		m.Rs.FieldNames[col.Name] = i
		switch col.Type {
		case value.IntType:
			m.Rs.Fields = append(m.Rs.Fields, mysql.NewField(as, s.Db, s.Db, 32, mysql.MYSQL_TYPE_LONG))
		case value.StringType:
			m.Rs.Fields = append(m.Rs.Fields, mysql.NewField(as, s.Db, s.Db, 200, mysql.MYSQL_TYPE_STRING))
		case value.NumberType:
			m.Rs.Fields = append(m.Rs.Fields, mysql.NewField(as, s.Db, s.Db, 32, mysql.MYSQL_TYPE_FLOAT))
		default:
			u.Warnf("Field type not known: %#v", col)
		}
		u.Debugf("added field: %#v", m.Rs.Fields[i])
	}

	u.Debugf("writeheaders: %#v", m.Rs.FieldNames)
	return nil
}

func (m *MysqlResultWriter) Finalize() error {
	if !m.wroteHeaders {
		err := m.WriteHeaders()
		if err != nil {
			return err
		}
	}

	// vals := make([]driver.Value, len(m.resp.Columns()))
	// for {

	// 	err := m.resp.Next(vals)
	// 	if err != nil && err == io.EOF {
	// 		break
	// 	} else if err != nil {
	// 		u.Error(err)
	// 		return err
	// 	}
	// 	u.Debugf("vals: %v", vals)
	// 	m.Rs.AddRowValues(vals)
	// }

	return nil
}