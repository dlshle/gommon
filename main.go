package main

import (
	"context"
	"fmt"
	"gommon/async"
	"gommon/http"
	"gommon/log"
	"os"
	"reflect"
	"time"

	// mongo
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	// mysql
	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	// "gommon/async"
	"gommon/deepcopy"
	"gommon/mysql"
	"gommon/performance"
	"gommon/timed"
	// orm
	"github.com/astaxie/beego/orm"

	lag "log"
)

type Unmarshaler interface {
	UnmarshalBSON([]byte) error
}

type User struct {
	Name string
	Age  int
}

type ICopy interface {
	Copy() interface{}
}

func (u *User) Copy() interface{} {
	return &User{u.Name, u.Age}
}

/*
for complicated structs, we might need a dedicated UnmarshalBSON implementation, but for simple structs, use pointer
type in Decode should work perfectly
e.g.
var user *User
...
err := cur.Decode(user)
*/
/*
func (user *User) UnmarshalBSON(b []byte) error {
  type Alias User
  bson.Unmarshal(b, (*Alias)(user))
  return nil
}
*/

func main() {
	func(run bool) {
		if !run {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:19950416@39.96.92.228:27017/test?authSource=admin"))

		defer func() {
			if err = client.Disconnect(ctx); err != nil {
				panic(err)
			}
		}()

		if err != nil {
			fmt.Println("error: ", err)
			return
		}
		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			fmt.Println("ping failed due to: ", err)
			return
		}

		collection := client.Database("test").Collection("users-go1")

		// find one
		func(run bool) {
			if !run {
				return
			}
			var result User
			// even though the struct uses Name, since the key is name in the collection, we need to use name as search key
			filter := bson.D{{"name", "Daniel"}}
			err := collection.FindOne(ctx, filter).Decode(&result)
			if err != nil {
				fmt.Println("find one error: ", err)
			} else {
				fmt.Printf("User find one result: %+v\n", result)
			}
		}(false)

		find := func(holder interface{}) []interface{} {
			res := make([]interface{}, 1)
			findOptions := options.Find()
			// findOptions.SetLimit(2)
			cur, err := collection.Find(ctx, bson.D{}, findOptions)
			if err != nil {
				fmt.Println("find failed due to ", err)
				return res
			}

			defer cur.Close(ctx)
			for cur.Next(ctx) {
				err := cur.Decode(holder)
				if err != nil {
					fmt.Println("cur decode error: ", err)
					return res
				}
				// TODO I need to append the new copy of the holder not the reference here...
				var cpy interface{}
				if _, ok := holder.(ICopy); ok {
					var copiable ICopy = holder.(ICopy)
					cpy = copiable.Copy()
					fmt.Printf("copied: %+v\n", cpy)
				} else {
					cpy = deepcopy.Copy(holder)
					fmt.Printf("deep copied: %+v\n", cpy)
				}
				res = append(res, cpy)
			}

			if err := cur.Err(); err != nil {
				fmt.Println("cur.Err() ", err)
			}
			return res
		}

		// find many
		func(run bool) {
			/*
			   if !run { return }
			   findOptions := options.Find()
			   findOptions.SetLimit(2)
			   cur, err := collection.Find(ctx, bson.D{}, findOptions)
			   if err != nil {
			     fmt.Println("find failed due to ", err)
			     return
			   }

			   defer cur.Close(ctx)
			   for cur.Next(ctx) {
			     var result User
			     err := cur.Decode(&result)
			     if err != nil {
			       fmt.Println("cur decode error: ", err)
			       return
			     }
			     fmt.Printf("result: %+v\n", result)
			   }

			   if err := cur.Err(); err != nil {
			     fmt.Println("cur.Err() ", err)
			   }
			*/

			usr := &User{}
			res := find(usr)
			for _, u := range res {
				fmt.Printf("%+v\n", u)
			}
			fmt.Println(res)
			fmt.Println(reflect.TypeOf(res[1]))
		}(true)

		// insert one
		func(run bool) {
			if !run {
				return
			}
			newUser := &User{"Xuri", 26}
			insertOneResult, err := collection.InsertOne(ctx, newUser)
			if err != nil {
				fmt.Println("insert failed ", err)
				return
			}
			fmt.Println("collection.InsertOne: ", insertOneResult)
		}(false)
	}(false)
	runWith(false, func() {
		dur := performance.Measure(func() {
			fmt.Println("promise and future test")
			p := async.NewAsyncPool("0", 10, 4)
			p.Start()
			p0 := p.Schedule(func() {
				time.Sleep(time.Second * 2)
				fmt.Println("2 seconds done")
			})

			p1 := p.Schedule(func() {
				time.Sleep(time.Second * 1)
				fmt.Println("1 seconds done")
			})

			go func() {
				p0.Wait()
				fmt.Println("f1")
			}()

			go func() {
				p0.Wait()
				fmt.Println("f2")
			}()

			go func() {
				time.Sleep(time.Millisecond * 1999)
				p0.Wait()
				fmt.Println("f wait 1.9 seconds")
			}()

			p0.Wait()
			fmt.Println("p0 waiting done")
			p1.Wait()
			fmt.Println("p1 waiting done")

			go func() {
				p0.Wait()
				fmt.Println("fx1")
			}()

			go func() {
				p0.Wait()
				fmt.Println("fx2")
			}()

			f0 := p.ScheduleComputable(func() interface{} {
				time.Sleep(1 * time.Second)
				return 1
			})

			go func() {
				fmt.Println("f00: ", f0.Get())
			}()

			go func() {
				fmt.Println("f01: ", f0.Get())
			}()

			go func() {
				time.Sleep(999 * time.Millisecond)
				fmt.Println("f01 999: ", f0.Get())
			}()

			fmt.Println("f0 result: ", f0.Get())
			p.Stop()
		})
		fmt.Println("task duration: ", dur)
		dur = performance.MeasureWithLog("100 async tasks", func() {
			pwaiters := make([]async.IPromise, 50)
			fwaiters := make([]async.IFuture, 50)
			p := async.NewAsyncPool("Hundred", 1024, 10)

			ptask := func() {
				time.Sleep(time.Second * 2)
			}

			ftask := func() interface{} {
				time.Sleep(time.Second * 1)
				return 5
			}

			for i := 0; i < 50; i++ {
				pwaiters = append(pwaiters, p.Schedule(ptask))
			}

			for i := 0; i < 50; i++ {
				fwaiters = append(fwaiters, p.ScheduleComputable(ftask))
			}

			for _, w := range pwaiters {
				if w != nil {
					w.Wait()
				}
			}

			for _, w := range fwaiters {
				if w != nil {
					w.Wait()
				}
			}

			p.Stop()
		})
	})

	func(run bool) {
		if !run {
			return
		}
		performance.MeasureWithLog("jobPool", func() {
			jobPool := timed.NewJobPool("t", 1)
			jobPool.ScheduleTimeoutJob(func() {
				fmt.Println("after 3 seconds")
			}, time.Second*3)
			uuid := jobPool.ScheduleAsyncIntervalJob(func() {
				fmt.Println("haha")
			}, time.Second*1)
			time.Sleep(time.Second * 5)
			jobPool.ScheduleAsyncTimeoutJob(func() {
				fmt.Printf("Remember me!?\n")
			}, time.Second*2)
			jobPool.CancelJob(uuid)
			time.Sleep(time.Second * 1)
		})
		timed.RunTimeout(func() {
			fmt.Println("global pool test")
		}, time.Second*2)
	}(false)

	func(run bool) {
		if !run {
			return
		}
		performance.MeasureWithLog("manyRequests", func() {
			request, _ := http.NewRequestBuilder().URL("https://www.baidu.com/home/msg/data/personalcontent?num=8&indextype=manht&_req_seqid=2361154953&asyn=1&t=1617529770282&sid=").Build()
			trArr := make([]*http.TrackableRequest, 1024)
			for i := 0; i < 1024; i++ {
				tr, _ := http.DoRequestAsync(request)
				trArr = append(trArr, tr)
			}
			for _, tr := range trArr {
				if tr != nil {
					tr.Response()
					fmt.Println(tr.Id() + " done")
				}
			}
		})
	}(false)
	runWith(false, func() {
		performance.MeasureWithLog("batchRequests", func() {
			request, _ := http.NewRequestBuilder().URL("https://www.baidu.com/home/msg/data/personalcontent?num=8&indextype=manht&_req_seqid=2361154953&asyn=1&t=1617529770282&sid=").Build()
			requests := make([]*http.Request, 100)
			for i, _ := range requests {
				requests[i] = request
			}
			responses := http.DoBatchRequest(requests)
			for _, resp := range responses {
				fmt.Println(resp)
			}
		})
	})
	runWith(false, func() {
		fmt.Println("=======================================================")
		db, err := sql.Open("mysql", "root:Lxr000518!@tcp(bj-cdb-l8bcf010.sql.tencentcdb.com:60856)/test?charset=utf8")
		if err != nil {
			fmt.Println("ERROR: ", err)
			return
		}
		defer db.Close()
		err = db.Ping()
		if err != nil {
			fmt.Println("Connection failed.")
			return
		}
		// email fn ln addr desc
		/*
			usrs:=[2][5] string{{"t1@1.1","ketty","lasty","addr1","desc1"},{"t2@2.2","rosee","last2","addr2","desc2"}}
			stmt,_:=db.Prepare("insert into USERS values (?,?,?,?,?)")
			for _,s:=range usrs{
				res, err := stmt.Exec(s[0],s[1],s[2],s[3],s[4])
				if err != nil {
					fmt.Println("ERROR: ", err)
				} else {
					fmt.Println("result: ", res)
				}
			}
		*/
		rows, _ := db.Query("select * from USERS") //获取所有数据

		var email, fn, ln, addr, desc string
		var age int
		for rows.Next() { //循环显示所有的数据
			rows.Scan(&email, &fn, &ln, &addr, &desc, &age)
			fmt.Printf("%s,%s,%s,%s,%s,%d\n", email, fn, ln, addr, desc, age)
		}
	})

	runWith(false, func() {
		type User struct {
			Id          int64
			Email       string
			FirstName   string
			LastName    string
			Address     string
			Description string
		}

		//1.连接数据库
		orm.RegisterDataBase("default", "mysql", "root:Lxr000518!@tcp(bj-cdb-l8bcf010.sql.tencentcdb.com:60856)/test")

		//2.注册表
		orm.RegisterModel(new(User))

		//3.生成表
		orm.RunSyncdb("default", false, true)

		o := orm.NewOrm()

		// --- INSERT
		newUser := &User{}
		newUser.FirstName = "Daniel"
		newUser.LastName = "Li"
		newUser.Address = "5003 176th ST SW APT E, Lynnwood, WA 98037"
		newUser.Description = "First record here of the user table."
		/*
			id, err := o.Insert(newUser)
			if err != nil {
				fmt.Println("Insert failed ", err)
				return
			}
			fmt.Println("inserted id: ", id)
		*/

		// --- READ
		var user User

		user.FirstName = "Daniel"

		err := o.Read(&user, "first_name")

		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		fmt.Println(user)
	})

	runWith(true, func() {
		type User struct {
			Id          int64 `orm:"pk;auto"`
			Email       string
			FirstName   string
			LastName    string
			Address     string
			Description string
		}

		m, err := mysql.NewSQLManager("bj-cdb-l8bcf010.sql.tencentcdb.com:60856", "root", "Lxr000518!", "test")
		if err != nil {
			fmt.Println("err init manager: ", err)
			return
		}
		m.RegisterORM(new(User))
		err = m.Start()
		if err != nil {
			fmt.Println("err start manager: ", err)
			return
		}
		var queryUser User
		queryUser.Id = 2
		fmt.Println(m.Read(&queryUser))

		/*
			manyUsers := []User{
				{FirstName: "a", LastName: "b"},
				{FirstName: "2", LastName: "2"},
				{FirstName: "v", LastName: "3"},
				{FirstName: "x", LastName: "4"},
				{FirstName: "z", LastName: "5"},
				{FirstName: "r", LastName: "6"},
				{FirstName: "e", LastName: "7"},
			}
			err = m.InsertMany(len(manyUsers), manyUsers)
			if err != nil {
				fmt.Println("insert many error: ", err)
				return
			}
		*/
		var users []User
		_, err = m.All(new(User), &users)
		if err != nil {
			fmt.Println("all error: ", err)
			return
		}
		fmt.Println(users)
	})
	runWith(true, LoggerTest)
}

func LoggerTest() {
	my := performance.MeasureWithLog("myLogger", func() {
		logger := log.NewDLogger(os.Stdout, log.LogDateTime, "[MyLogger]", false)
		for i := 0; i < 1000; i++ {
			logger.Info("")
		}
	})
	os := performance.MeasureWithLog("osLogger", func() {
		lagger := lag.New(os.Stdout, "[OsLogger]", lag.Ldate|lag.Ltime)
		for i := 0; i < 1000; i++ {
			lagger.Println("")
		}
	})
	fmt.Println("my: ", my, "os: ", os)
}

func runWith(run bool, executor func()) {
	if !run {
		return
	}
	executor()
}

func buildQueryFilter(filterMap map[string]interface{}) interface{} {
	filter := bson.M{}
	data, err := bson.Marshal(filter)
	if err != nil {
		fmt.Println("unable to convert filter due to ", err)
		return nil
	}
	err = bson.Unmarshal(data, filter)
	if err != nil {
		fmt.Println("unable to unmarshal due to ", err)
		return nil
	}
	return filter
}
