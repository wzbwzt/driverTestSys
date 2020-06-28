# 项目概述
模拟驾考系统；随机生成指定数量的学员进行并发驾考，随机给打定分数，小于指定值的淘汰掉，继续下一学员考试；开始完成后将学员分数写入到数据库，查询时使用二级缓存；优先从redis中查找，没有的话先写入到redis中，再从mysql中查找

# 需求点
### 生成随机的数量的不同考生姓名，考场签到，并名字丢入管道；
### 考场只有5个车道，所以最多供5个人同时考试；
### 考生按签到顺序依次考试，给予考生10%的违规几率；
### 考场巡视人员每1秒钟巡视一次，发现违规的清出考场，否则输出考场时序良好(并发执行)；
### 所有考试者考完后，向MySQL数据库录入考试成绩(并发执行)；
### 成绩录入完毕通知考生，考生查阅自己的成绩(首先是从redis缓存中读取数据，如果没有的话，再从Mysql中读取，并写入到redis中)；
### 查询成绩时开读写锁，写时不可以读，允许多读；
### 再次查询成绩使用Redis缓存（二级缓存）；

# 技术栈
### 管道并发
### MySQL-Redis连接池
### MySQL-Redis二级缓存
### 通用的数据库工具的封装
### 使用go module进行依赖版本管理
### 类库封装和复用