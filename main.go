package sqlbee

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type SqlBeeE struct {
	SqlBeeCommon
	Method        SqlBeeE_MethodType
	Insert_values map[string]any
	Insert_lastId int //插入操作之后返回的新数据条的id
	Update_values map[string]any
	Update_ids    []int //更新操作之后返回的受影响的所有数据条的id
}

// “安全材料”，在update的时候用
//
// 删除map里特定的key们（有时候map里可能包含例如id等filter所用的值），且默认删除“id”这个键
//
// ‼️ 因为map是指针类型，所以该操作会更改map的值，请留意该操作的使用位置
func (r *SqlBeeE) safeMaterial(_map map[string]any, _keys ...string) {
	delete(_map, "id")
	for _, v := range _keys {
		delete(_map, v)
	}
}

// 当数据库完成一次操作run后，取回的内容
//
// 目前只能作用于 Insert, Update（update是空白内容，还没做！）
func (r *SqlBeeE) getAfterRun() []map[string]any {
	if r.Method != Insert && r.Method != Update {
		log.Fatalln("[SqlBee🐝]: this method only works for Insert & Update")
	}
	if !r.RunStatus {
		log.Fatalln("[SqlBee🐝]: this method only works after run() has been applied.")
	}
	var beeQ SqlBeeQ
	_filter := make(map[string]any)
	if r.Method == Insert {
		_filter["id"] = strconv.Itoa(r.Insert_lastId)
	}
	if r.Method == Update {
		_filter["id"] = strconv.Itoa(r.Update_ids[0]) //目前只能做单个id
	}
	beeQ.model(r.Model)
	beeQ.filter(_filter)
	bag := beeQ.run()
	return bag
}

// 返回 (用于数据库储存的) 当前时间
func (r *SqlBeeE) current_time_for_writeIn() string {
	return time.Now().Format("2006-01-02T15:04:05 -07:00:00")
}

// 返回 (用于数据库储存的) 时间
func (r *SqlBeeE) time_for_writeIn(_time time.Time) string {
	return _time.Format("2006-01-02T15:04:05 -07:00:00")
}

// 指定插入语句
func (r *SqlBeeE) insert(values map[string]any) {
	if r.Method != None {
		log.Fatalln("[SqlBee🐝]: For Insert/Delete/Update methods you can only use one of them for a run.")
	} else {
		r.Insert_values = values
		r.Method = Insert
	}
}

// 指定更新语句
func (r *SqlBeeE) update(values map[string]any) {
	if r.Method != None {
		log.Fatalln("[SqlBee🐝]: For Insert/Delete/Update methods you can only use one of them for a run.")
	} else {
		r.Update_values = values
		r.Method = Update
	}
}

// 指定删除
func (r *SqlBeeE) delete() {
	if r.Method != None {
		log.Fatalln("[SqlBee🐝]: For Insert/Delete/Update methods you can only use one of them for a run.")
	} else {
		r.Method = Delete

	}
}

// 运行操作 run ⏩
func (r *SqlBeeE) run() int {
	if r.Model == nil {
		log.Fatalln("[SqlBee🐝]: Fatal, you must use .model(Model{}) before run()")
	}
	//
	bee, err := SqlBeeInstance()
	if err != nil {
		log.Fatalln(err)
	}
	defer bee.Close()
	// ⬇️ 拼凑执行语句 start
	exec := ""
	//
	switch r.Method {
	case Insert:
		insert_values := ""
		insert_keys := ""
		// ？可以对key做一个判断
		for k, v := range r.Insert_values {
			//以字符串格式''概括，留给数据库自行转化类型
			insert_values += fmt.Sprintf("'%v',", v)
			insert_keys += k + ","
		}
		insert_keys = strings.TrimSuffix(insert_keys, ",")
		insert_values = strings.TrimSuffix(insert_values, ",")
		//最后的执行sql语句
		exec += "INSERT INTO " + r.Table + " (" + insert_keys + ") VALUES (" + insert_values + ")"
		fmt.Println(exec)
	case Update:
		set_kvs := ""
		set_wheres := r.SQL_Semantics_WHERE(r.Filter)
		for k, v := range r.Update_values {
			//以字符串格式''概括，留给数据库自行转化类型
			set_kvs += fmt.Sprintf("%s='%v',", k, v)
		}
		set_kvs = strings.TrimSuffix(set_kvs, ",")
		exec += "UPDATE " + r.Table + " SET " + set_kvs + set_wheres
	case Delete:
		if len(r.Filter) == 0 {
			log.Fatalln("[SqlBee🐝]: DELETE method must have a valid .filter(_filter) where _filter has at least one element.")
		}
		exec += "DELETE FROM " + r.Table + r.SQL_Semantics_WHERE(r.Filter)
	default:
		log.Fatalln("[SqlBee🐝]: A valid Method must be given by using method().")
	}
	// ⬆️ 拼凑执行语句 end
	//
	feedback, err := bee.Exec(exec)
	if err != nil {
		fmt.Println("exec failed, ", err)
		return -1
	}
	var worked_id int64
	if r.Method == Insert {
		worked_id, _ = feedback.LastInsertId()
		r.Insert_lastId = int(worked_id)
	}
	if r.Method == Update {
		worked_id, _ = feedback.RowsAffected()
	}
	if r.Method == Delete {
		worked_id, _ = feedback.RowsAffected()
	}
	r.RunStatus = true
	return int(worked_id)
}

type SqlBeeQ struct {
	SqlBeeCommon
	Order_by string
	Limit    int
}

// 指定排序
func (r *SqlBeeQ) order_by(_order_by string) {
	r.Order_by = _order_by
}

// 指定获得的数量
func (r *SqlBeeQ) limit(_limit int) {
	r.Limit = _limit
}

// 运行查询 run ⏩
func (r SqlBeeQ) run() []map[string]any {

	if r.Model == nil {
		log.Fatalln("[SqlBee🐝]: Fatal, you must use .model(Model{}) before run()")
	}

	table := r.Table
	filter := r.Filter
	order_by := r.Order_by
	limit := r.Limit

	//默认按id排序
	if len(order_by) == 0 {
		order_by = "id"
	} else {
		// 倒叙设置
		if _desc := strings.HasPrefix(order_by, "-"); _desc {
			order_by = strings.TrimPrefix(order_by, "-")
			order_by += " DESC "
		}
	}
	//默认取10条
	//struct的int默认设置为0
	if limit == 0 {
		limit = 10
	}
	//
	bee, err := SqlBeeInstance()
	if err != nil {
		log.Fatalln(err)
	}
	defer bee.Close()
	//
	// ⬇️ 拼凑查询语句 start
	query := "SELECT * FROM "
	//
	query += table
	//
	//
	query += r.SQL_Semantics_WHERE(filter)
	//
	query += " ORDER BY " + order_by
	//
	query += " LIMIT " + strconv.Itoa(limit)
	//
	query += ";"
	// ⬆️ 拼凑查询语句 end
	rows, _ := bee.Query(query)
	// ********************************
	// ⬇️ 准备 即将返回给客户端的一个map的数组

	bag := SqlBee_ScanToMapBag(rows, r.Model)

	defer rows.Close()

	r.RunStatus = true

	return bag
}
