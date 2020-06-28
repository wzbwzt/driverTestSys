package utils

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)
/*考试成绩*/
type ExamScore struct {
	Id    int    `db:"id"`
	Name  string `db:"name"`
	Score int    `db:"score"`
}

var (
	chNames = make(chan string, 100)
	//随机数互斥锁（确保GetRandomInt不能被并发访问）
	randomMutex sync.Mutex
	//姓氏
	familyNames = []string{"赵", "钱", "孙", "李", "周", "吴", "郑", "王", "冯", "陈", "楚", "卫", "蒋", "沈", "韩", "杨", "张", "欧阳", "东门", "西门", "上官", "诸葛", "司徒", "司空", "夏侯"}
	//辈分（欧阳：宗的永其光...）
	middleNamesMap = map[string][]string{}
	//名字
	lastNames = []string{"春", "夏", "秋", "冬", "风", "霜", "雨", "雪", "木", "禾", "米", "竹", "山", "石", "田", "土", "福", "禄", "寿", "喜", "文", "武", "才", "华"}
	//数据库读写锁
	dbMutext sync.RWMutex
	//mysqlDB
	mysqlDB *sqlx.DB
	//redis连接
	RedisPool *redis.Pool
)
/*处理错误：有错误时暴力退出*/
func HandlerError(err error, when string) {
	if err != nil {
		fmt.Println(when, err)
		os.Exit(1)
	}
}
/*初始化姓氏和对应的辈分*/
func init() {
	for _, x := range familyNames {
		if x != "欧阳" {
			middleNamesMap[x] = []string{"德", "惟", "守", "世", "令", "子", "伯", "师", "希", "与", "孟", "由", "宜", "顺", "元", "允", "宗", "仲", "士", "不", "善", "汝", "崇", "必", "良", "友", "季", "同"}
		} else {
			middleNamesMap[x] = []string{"宗", "的", "永", "其", "光"}
		}
	}
}
/*获取[start,end]范围内的随机数*/
func GetRandomInt(start, end int) int {
	randomMutex.Lock()
	<-time.After(1 * time.Nanosecond)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := start + r.Intn(end-start+1)
	randomMutex.Unlock()
	return n
}
/*获得随机姓名*/
func GetRandomName() (name string) {
	familyName := familyNames[GetRandomInt(0, len(familyNames)-1)]
	middleName := middleNamesMap[familyName][GetRandomInt(0, len(middleNamesMap[familyName])-1)]
	lastName := lastNames[GetRandomInt(0, len(lastNames)-1)]
	return familyName + middleName + lastName
}

/*mysql初始化*/
func InitMysqlDB(){
	db, err := sqlx.Connect("mysql", "root:123123@tcp(192.168.241.129:3306)/go_demo")
	HandlerError(err, `sqlx.Connect("mysql", "root:123123@tcp(192.168.241.129:3306)/go_demo")`)
	mysqlDB=db
}
/*Mysql关闭*/
func MysqlClose(){
	mysqlDB.Close()
}

/*将全员考试成绩单写入MySQL数据库*/
func WriteScore2Mysql(scoreMap map[string]int) {
	//锁定为写模式，写入期间不允许读访问
	dbMutext.Lock()
	for name, score := range scoreMap {
		_, err := mysqlDB.Exec("insert into score(name,score) values(?,?);", name, score)
		HandlerError(err, `db.Exec("insert into score(name,score) values(?,?);", name, score)`)
		fmt.Println("插入成功！")
	}
	fmt.Println("成绩录入完毕！")
	//解锁数据库，开放查询
	dbMutext.Unlock()
}

/*
通用Mysql查询工具
tableName	要查询的表名
argsMap		查询条件集合
dest		查询结果存储地址
*/
func QueryFromMysql(tableName string,argsMap map[string]interface{},dest interface{}) (err error) {
	fmt.Println("QueryScoreFromMysql...")
	//写入期间不能进行数据库读访问
	dbMutext.RLock()
	selection := ""
	values := make([]interface{}, 0)
	for col,value := range argsMap{
		selection += (" and "+col+"=?")
		values = append(values, value)
	}
	selection = selection[4:]
	sql := "select * from "+tableName+" where "+selection;

	err = mysqlDB.Select(dest, sql, values...)
	if err != nil {
		fmt.Println(err, `db.Select(&examScores, "select * from score where name=?;", name)`)
		return
	}
	dbMutext.RUnlock()
	return
}

/*获取redis连接（连接池）*/
func GetRedisConn()redis.Conn{
	if RedisPool == nil {
		RedisPool=&redis.Pool{
			MaxActive: 100,//最大连接数
			MaxIdle: 10,//最大闲置数
			IdleTimeout: 10,//闲置的超时时间10s
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "192.168.241.129:6379")
			},
		}
	}
	return RedisPool.Get()
}
/*redis连接关闭*/
func RedisClose(){
	RedisPool.Close()
}

/*redis 读写相关操作*/
func DoRedis(cmd string)(score int,err error){
	conn := GetRedisConn()
	split := strings.Split(cmd, " ")
	command:= split[0]
	commandContent:= make([]interface{},0)
	for _,arg:= range split[1:]{
		commandContent = append(commandContent, arg)
	}
	if command == "get" {
		reply, err := conn.Do(command, commandContent...)
		score,err=redis.Int(reply,err)
		return score,err
	}else {
		_, err := conn.Do(command, commandContent...)
		return 0,err
	}
}




