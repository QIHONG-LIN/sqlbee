package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
)

type SqlBeeWorker struct {
	does           SqlBeeFilter
	model          any
	table_name_sql string
}

// Methods being used after filter() like filter, exclude, order_by, get, delete they're the members of SqlBeeFilter.
// filter, exclude, order_by will return SqlBeeFilter that allow you to continue to use filter & exclude
// but get, delete serve as the end.
type SqlBeeFilter struct {
	exclude_sql     string
	order_by_sql    string
	filter_sql      string
	all_filter_mode bool // if this is true then you cannot use filter()
	table_name_sql  string
}

// Set up SQL of how to find rows, it help to select all rows in a table.
//
// Remark: if you use all() then filter() will loss its effect.
func (m *SqlBeeFilter) all() *SqlBeeFilter {

	m.filter_sql = ""
	m.all_filter_mode = true
	return m
}

// Set up SQL of how to find rows, it forms a WHERE query string.
func (m *SqlBeeFilter) filter(_map map[string]any) *SqlBeeFilter {

	if m.all_filter_mode {
		return m
	}

	_filter := ""

	if len(_map) > 0 {
		// this secure that WHERE keyword only writes in once
		if len(m.filter_sql) == 0 && len(m.exclude_sql) == 0 {
			_filter = " WHERE "
		}
		_filter += SqlBee_SQL_Semantics_WHERE(_map, "=")
	}

	m.filter_sql += _filter

	return m
}

// Set up SQL of how to exclude some rows, it forms a WHERE query string.
func (m *SqlBeeFilter) exclude(_map map[string]any) *SqlBeeFilter {

	_exclude := ""

	if len(_map) > 0 {

		// this secure that WHERE keyword only writes in once
		if len(m.filter_sql) == 0 && len(m.exclude_sql) == 0 {
			_exclude = " WHERE "
		}

		_exclude += SqlBee_SQL_Semantics_WHERE(_map, "<>")
	}

	m.exclude_sql = _exclude

	return m
}

// Set up how you want to order the rows in query.
//
// A common use is .order_by("id") or you can use .order_by("-id") to do descending.
func (m *SqlBeeFilter) order_by(_order string) *SqlBeeFilter {

	if len(_order) == 0 {
		return m
	}

	// descending
	if _desc := strings.HasPrefix(_order, "-"); _desc {
		_order = strings.TrimPrefix(_order, "-")
		_order += " DESC "
	}

	m.order_by_sql = " ORDER BY " + _order

	return m
}

// !! End method [put it in the end of methods chain]
//
// Here the final SQL command will be formed and runned, it returns bags.
//
// the SQL segments are from SqlBeeFilter.
func (m *SqlBeeFilter) get() {

	exec := "SELECT * FROM " + m.table_name_sql

	exec += m.filter_sql + m.exclude_sql + m.order_by_sql

	fmt.Println(exec)

}

// [End Method] delete()
//
// Delete the row in database in respect to the given struct.
//
// A common application is to use "user.delete()" to delete a certain row in term of 'id' only.
func (m *SqlBeeWorker) delete() {

	it := m.Struct_To_Map()
	exec := ""

	_filter := make(map[string]any)
	_filter["id"] = it["id"]

	exec += "DELETE FROM " + m.table_name_sql + " WHERE " + SqlBee_SQL_Semantics_WHERE(_filter, "=")

}

// [End Method] save()
//
// Save a row in database for the given struct if it has 'id'==0;
//
// Or update the row correspond to its 'id' if 'id'>0.
func (m *SqlBeeWorker) save() {

	/*
		instance, err := SqlBeeDbInstance()
		if err != nil {
			fmt.Println(err)
		}

		defer instance.Close()*/

	data_map := m.Struct_To_Map()

	exec := "" //SQL command

	if data_map["id"].(int) <= 0 {
		//insert
		insert_values := ""
		insert_keys := ""

		delete(data_map, "id")

		for k, v := range data_map {
			insert_values += fmt.Sprintf("'%v',", v)
			insert_keys += k + ","
		}

		insert_keys = strings.TrimSuffix(insert_keys, ",")
		insert_values = strings.TrimSuffix(insert_values, ",")

		exec += "INSERT INTO " + m.table_name_sql + " (" + insert_keys + ") VALUES (" + insert_values + ")"

	} else {
		//update
		set_kvs := ""

		_id_filter := map[string]any{"id": data_map["id"]}
		set_wheres := SqlBee_SQL_Semantics_WHERE(_id_filter, "=")

		for k, v := range data_map {
			set_kvs += fmt.Sprintf("%s='%v',", k, v)
		}

		set_kvs = strings.TrimSuffix(set_kvs, ",")

		exec += "UPDATE " + m.table_name_sql + " SET " + set_kvs + set_wheres

	}

	fmt.Println(exec)

}

// Convert bee's struct to map, and this map can be used to do business with the database. (i.e., save, update...)
func (m *SqlBeeWorker) Struct_To_Map() map[string]any {
	ref_t := reflect.TypeOf(m.model)
	ref_v := reflect.ValueOf(m.model)
	KeyValueCope := make(map[string]any)
	for i := 0; i < ref_t.NumField(); i++ {
		//unCapitalizeFirstLetter keeps the same name form as in database.
		var await_value any
		switch ref_t.Field(i).Type.Kind() {
		case reflect.Int:
			await_value = int(ref_v.Field(i).Int())
		default:
			await_value = ref_v.Field(i).String()
		}
		KeyValueCope[unCapitalizeFirstLetter(ref_t.Field(i).Name)] = await_value
	}
	fmt.Println(KeyValueCope)
	return KeyValueCope
}

// SummonBeeFrom - [independent function][entrance]
//
// You use SummonBeeFrom(xxx{}) to summon a bee that works for xxx struct
// and access to all methods in sqlbee package!
//
// This is the entrance of the package.
//
// Here is a common example:
//
//	bee := SummonBeeFrom(User{})
//	results := bee.filter(map1).exclude(map2).order_by("-id").get()
//	one_result := bee.filter(map1).get()
//
// You can import a filled struct and save it to database:
//
//	bee := SummonBeeFrom(User{name:"qihong,lin"})
//	bee.save()
func SummonBeeFrom(_beehive any) SqlBeeWorker {

	table_name := unCapitalizeFirstLetter(getStructName(_beehive, false))

	//for bee.xxx()
	var worker_bee SqlBeeWorker
	worker_bee.model = _beehive
	worker_bee.table_name_sql = table_name

	//for bee.does.xxx()
	var action_bee SqlBeeFilter
	action_bee.table_name_sql = table_name
	worker_bee.does = action_bee

	return worker_bee
}

// An Instance of connection to database.
func SqlBeeDbInstance() (*sql.DB, error) {

	// Set up the database source string.
	setting := ReadSqlBeeDbSetting()

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local", setting.SqlBeeDb_username, setting.SqlBeeDb_password, setting.SqlBeeDb_host, setting.SqlBeeDb_port, setting.SqlBeeDb_dbname)

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

// Read from sqlbee_setting.json into a struct.
func ReadSqlBeeDbSetting() SqlBeeDbSetting {
	var setting SqlBeeDbSetting
	fileContent, err := os.Open("sqlbee_setting.json")
	if err != nil {
		log.Fatal(err)
		return setting
	}
	defer fileContent.Close()
	fmt.Println("✅ load sqlbee_setting.json successfully.")
	byteResult, _ := ioutil.ReadAll(fileContent)
	fmt.Println("err:", json.Unmarshal(byteResult, &setting))
	return setting
}

type SqlBeeDbSetting struct {
	SqlBeeDb_username string `json:"SqlBeeDb_username"`
	SqlBeeDb_password string `json:"SqlBeeDb_password"`
	SqlBeeDb_host     string `json:"SqlBeeDb_host"`
	SqlBeeDb_dbname   string `json:"SqlBeeDb_dbname"`
	SqlBeeDb_port     int    `json:"SqlBeeDb_port"`
}

func main() {

	_map := make(map[string]any)

	_map["city"] = "shanghai"
	_map["city2"] = "shanghai2"

	_map2 := make(map[string]any)

	_map2["gender"] = "female"

	var user User

	user.Age = 18
	user.Name = "qihong"

	bee := SummonBeeFrom(user)
	//bee.does.all().exclude(_map2).order_by("-id").get()
	bee.save()

	user.Id = 1
	bee2 := SummonBeeFrom(user)
	bee2.delete()

	/*
		var user User
		_map := make(map[string]any)
		//
		user.SqlBeeWorker().filter(_map).exclude(_map).order_by("-id").get()
		//user.SqlBeeWorker().filter(_map).get()
		//user.SqlBeeWorker().filter(_map).delete()
		//
		user.save()*/
}

// this function helps to form WHERE query string.
//
// 'relation' is the operator such as "=", ">", "<>", ...
func SqlBee_SQL_Semantics_WHERE(_filter map[string]any, relation string) string {

	query := ""

	if len(_filter) > 0 {

		for k, v := range _filter {

			//以字符串格式''概括，留给数据库自行转化类型
			query += k + relation + fmt.Sprintf("'%v'", v) + " AND "

		}
	}
	return strings.TrimSuffix(query, "AND ")
}

func getStructName(myvar interface{}, ignorePtr bool) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		if ignorePtr {
			return t.Elem().Name()
		}
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}

// unCapitalizeFirstLetter
//
// i.e., transform "StarColumn" to "starColumn"
func unCapitalizeFirstLetter(str string) string {
	str_part1 := str[0:1]
	str_part2 := str[1:]
	str = strings.ToLower(str_part1) + str_part2
	return str
}

// /
type User struct {
	Id     int
	Name   string
	Age    int
	Nation string
}
