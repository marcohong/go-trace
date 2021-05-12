package orm

import (
	"go-trace/trace"
	"strings"

	"gorm.io/gorm"
)

const (
	gormSpanKey        = "__gorm_span"
	callBackBeforeName = "opentracing:before"
	callBackAfterName  = "opentracing:after"
)

func before(db *gorm.DB) {
	tr, ok := trace.StartSpanFromContextV2(db.Statement.Context, "gorm")
	if !ok {
		return
	}
	tr.SetTag(trace.Tag(trace.TagPeerService, "database"))
	tr.SetTag(trace.Tag(trace.TagSpanKind, "client"))
	tr.SetTag(trace.Tag(trace.TagComponent, "db/gorm"))
	tr.SetTag(trace.Tag(trace.TagDBType, "sql"))
	db.InstanceSet(gormSpanKey, tr)
}

func after(db *gorm.DB) {
	val, ok := db.InstanceGet(gormSpanKey)
	if !ok {
		return
	}
	tr, ok := val.(trace.Tracer)
	if !ok {
		return
	}
	tr.SetTag(trace.Tag(trace.TagDBStatement, strings.ToLower(db.Statement.SQL.String())))

	if db.Error != nil {
		tr.SetLog(trace.LogBool(trace.TagError, true), trace.LogString(trace.LogMessage, db.Error.Error()))
		tr.Finish(&db.Error)
		return
	}
	tr.Finish(nil)
}

// OpentracingPlugin .
type OpentracingPlugin struct{}

// Name returns the name of the plugin
func (op *OpentracingPlugin) Name() string {
	return "opentracingPlugin"
}

// Initialize init OpentracingPlugin
func (op *OpentracingPlugin) Initialize(db *gorm.DB) (err error) {
	// start before
	db.Callback().Create().Before("gorm:before_create").Register(callBackBeforeName, before)
	db.Callback().Query().Before("gorm:query").Register(callBackBeforeName, before)
	db.Callback().Delete().Before("gorm:before_delete").Register(callBackBeforeName, before)
	db.Callback().Update().Before("gorm:setup_reflect_value").Register(callBackBeforeName, before)
	db.Callback().Row().Before("gorm:row").Register(callBackBeforeName, before)
	db.Callback().Raw().Before("gorm:raw").Register(callBackBeforeName, before)

	// finish after
	db.Callback().Create().After("gorm:after_create").Register(callBackAfterName, after)
	db.Callback().Query().After("gorm:after_query").Register(callBackAfterName, after)
	db.Callback().Delete().After("gorm:after_delete").Register(callBackAfterName, after)
	db.Callback().Update().After("gorm:after_update").Register(callBackAfterName, after)
	db.Callback().Row().After("gorm:row").Register(callBackAfterName, after)
	db.Callback().Raw().After("gorm:raw").Register(callBackAfterName, after)
	return
}
