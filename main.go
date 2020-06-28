package main

import (
	"driverTestSys/utils"
	"fmt"
	"strconv"
	"sync"
	"time"
)

var(
	chNames = make(chan string, 100)
	examers = make([]string, 0)
	//信号量，只有5条车道
	chLanes = make(chan int, 5)
	//违纪者
	chFouls = make(chan string, 100)
	//考试成绩
	scoreMap = make(map[string]int)
	wg sync.WaitGroup
)

/*巡考逻辑*/
func Patrol() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case name := <-chFouls:
			fmt.Println(name, "考试违纪!!!!! ")
		default:
			fmt.Println("考场秩序良好")
		}
		<-ticker.C
	}
}

/*考试逻辑*/
func TakeExam(name string) {
	chLanes <- 123
	fmt.Println(name, "正在考试...")
	//记录参与考试的考生姓名
	examers = append(examers, name)
	//生成考试成绩
	score := utils.GetRandomInt(0, 100)
	scoreMap[name] = score
	if score < 10 {
		chFouls <- name
		//fmt.Println(name, "考试违纪！！！", score)
	}
	//考试持续5秒
	<-time.After(5 * time.Second)
	<-chLanes
	//wg.Done()
}

/*二级缓存查询成绩*/
func QueryScore(name string) {
	//score, err := utils.QueryScoreFromRedis(name)
	cmd:="get "+name
	score, err := utils.DoRedis(cmd)
	if err != nil {
		scores := make([]utils.ExamScore, 0)
		argsMap := make(map[string]interface{})
		argsMap["name"] = name
		err = utils.QueryFromMysql("score", argsMap, &scores)
		utils.HandlerError(err,`utils.QueryFromMysql("score", argsMap, &scores)`)
		fmt.Println("Mysql成绩：", name, ":", scores[0].Score)
		/*将数据写入Redis*/
		cmd:="set "+name+" "+strconv.Itoa(scores[0].Score)
		_, err := utils.DoRedis(cmd)
		if err != nil {
			fmt.Println("set to redis err;err:",err)
			return
		}
		//utils.WriteScore2Redis(name, scores[0].Score)
	} else {
		fmt.Println("Redis成绩：", name, ":", score)
	}
}


func main() {
	for i := 0; i < 10; i++ {
		chNames <- utils.GetRandomName()
	}
	close(chNames)

	/*巡考*/
	go Patrol()

	/*考生并发考试*/
	for name := range chNames {
		wg.Add(1)
		go func(name string) {
			TakeExam(name)
			wg.Done()
		}(name)
	}
	wg.Wait()
	fmt.Println("考试完毕！")
	//初始化mysql
	utils.InitMysqlDB()
	defer utils.MysqlClose()
	//关闭redis连接
	defer utils.RedisClose()

	//录入成绩
	wg.Add(1)
	go func() {
		utils.WriteScore2Mysql(scoreMap)
		wg.Done()
	}()
	//故意给一个时间间隔，确保WriteScore2DB先抢到数据库的读写锁
	<-time.After(1 * time.Second)

	/*考生查询成绩*/
	for _, name := range examers {
		wg.Add(1)
		go func(name string) {
			QueryScore(name)
			wg.Done()
		}(name)
	}
	<-time.After(1 * time.Second)
	for _, name := range examers {
		wg.Add(1)
		go func(name string) {
			QueryScore(name)
			wg.Done()
		}(name)
	}

	wg.Wait()
	fmt.Println("END")
}
