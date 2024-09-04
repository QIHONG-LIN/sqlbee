package sqlbee

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

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

func JsonToMap(jsonStr string) (map[string]any, error) {
	m := make(map[string]any)
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		fmt.Printf("Unmarshal with error: %+v\n", err)
		return nil, err
	}

	for k, v := range m {
		fmt.Printf("%v: %v\n", k, v)
	}

	return m, nil
}

func getPostAll(c *gin.Context) map[string]any {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Println("getPost: 读取body失败")
	}
	result, err := JsonToMap(string(body))
	if err != nil {
		fmt.Println("getPost: JsonToMap失败")
	}
	return result

}
