package main

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// SqlBeeE的method type (枚举)
type SqlBeeE_MethodType int

const (
	// 第一个元素为None，才能让之后初始化的时候不会自动选择方法，这样方法内的报错才能生效
	None SqlBeeE_MethodType = iota
	Insert
	Delete
	Update
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

// 层次结构为：
//
// SqlBeeHelper -> SqlBeeCommon -> SqlBee[?]
type SqlBeeCommon struct {
	SqlBeeHelper
	Table     string
	Model     any
	Filter    map[string]any
	RunStatus bool //是否run()过？
}

// 指定SqlBeeModel模型
//
/*
【关于模型】

1.必须在struct的第一个位置填入 SqlBeeModel string `table_name:"xxxx"`，其中xxxx为数据库的实际表名

这代表着一个struct注册为了 SqlBeeModel，其将被允许用在bee.model(struct_name{})

2. 特别注意，此处的struct必须包含所有数据库实际的字段，且名称完全一致（首字母大小写可忽略），但可以包含一些数据库没有的字段的额外Field
。例如数据库字段为 [id, name, age]，那么struct可以为 [SqlBeeModel, Id, Name, Age, Location, ...]
*/
func (r *SqlBeeCommon) model(_model any) {

	rv := reflect.TypeOf(_model)
	if rv.Field(0).Name != "SqlBeeModel" {
		log.Fatalln("[SqlBee🐝]: Fatal, check if 'SqlBeeModel' is in your Model and should be the first property.")
	}

	r.Model = _model

	//同时指定表名
	table := fmt.Sprintf("%v", rv.Field(0).Tag.Get("table_name"))
	r.Table = table
}

// 指定filter
func (r *SqlBeeCommon) filter(_filter map[string]any) {
	r.Filter = _filter
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

// 连接数据库的实例
func SqlBeeInstance() (*sql.DB, error) {

	// Set up the database source string.
	setting := getGoSetting()

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local", setting.Database_username, setting.Database_password, setting.Database_host, setting.Database_port, setting.Database_dbname)

	// Create a database handle and open a connection pool.
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Check if our connection is alive.
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
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

// 层次结构为：
//
// SqlBeeHelper -> SqlBeeCommon -> SqlBee[?]
//
// SqlBeeHelper是最顶层的配置
type SqlBeeHelper struct {
}

// 为 WHERE 解析语义（包含WHERE关键词本身）
//
// 返回例如 " WHERE name = '张三' AND city = 'Shanghai' "
func (r *SqlBeeHelper) SQL_Semantics_WHERE(_filter map[string]any) string {
	query := ""
	if len(_filter) > 0 {
		query += " WHERE "

		for k, v := range _filter {

			//以字符串格式''概括，留给数据库自行转化类型
			query += k + "=" + fmt.Sprintf("'%v'", v) + " AND "

		}
	}
	return strings.TrimSuffix(query, "AND ")
}

// 将数据库读取的rows scan 到一个map的bag里
//
// 需要传入一个对应数据类型的struct
func SqlBee_ScanToMapBag[T any](_rows *sql.Rows, _struct T) []map[string]any {
	//
	bag := make([]map[string]any, 0)
	//拿到表里的所有字段名
	cols, err := _rows.Columns()
	if err != nil {
		log.Fatalln(err)
	}
	//准备一个存放指针的数组
	scan_pointers := make([]any, len(cols))
	//准备一个byte数组
	scanned_values := make([][]byte, len(cols))
	//将byte数组内的空byte对象的指针放入
	for i := range scanned_values {
		scan_pointers[i] = &scanned_values[i]
	}
	//拿到struct对象
	//设置一个field的 name:type 键值对
	ref := reflect.TypeOf(_struct)
	correspond_nameTypeCope := make(map[string]reflect.Kind)
	for i := 1; i < ref.NumField(); i++ {
		//去掉model的首字母大写
		correspond_nameTypeCope[unCapitalize(ref.Field(i).Name)] = ref.Field(i).Type.Kind()
	}
	for _rows.Next() {
		//对指针数组赋值
		err = _rows.Scan(scan_pointers...)
		if err != nil {
			log.Fatalln(err)
		}
		//准备一个返回 🔙
		_ready_map := make(map[string]any)
		//
		// 注意 --> scanned_values 和 cols 的顺序一致；
		// 我们用cols的值作为key去取correspond_nameTypeCope的type；
		//
		for i, v := range scanned_values {
			_ready_map[cols[i]] = string(v)
			_possible_type := correspond_nameTypeCope[cols[i]]
			//根据记录好的数据类型将string转为any
			switch _possible_type {
			//转化为int
			case reflect.Int:
				to_int, err := strconv.Atoi(string(v))
				if err == nil {
					_ready_map[cols[i]] = to_int
				} else {
					//处理parsing "": invalid syntax
					_ready_map[cols[i]] = nil
				}

			}
		}
		bag = append(bag, _ready_map)
	}

	return bag

}

// 将字符首字母大写变为小写
//
// 这是为了把go的model形态转化为数据库/客户端json
func unCapitalize(str string) string {
	str_part1 := str[0:1]
	str_part2 := str[1:]
	str = strings.ToLower(str_part1) + str_part2
	return str
}
