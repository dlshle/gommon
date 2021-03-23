package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"mongoClient/async"
	"mongoClient/deepcopy"
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




	fmt.Println("promise and future test")
	p := async.NewAsyncPool(10, 4)
	p.Start()
	p0 := p.Schedule(func() {
		time.Sleep(time.Second * 2)
		fmt.Println("2 seconds done")
	})

	p1 := p.Schedule(func() {
		time.Sleep(time.Second * 1)
		fmt.Println("1 seconds done")
	})

	p0.Wait()
	fmt.Println("p0 waiting done")
	p1.Wait()
	fmt.Println("p1 waiting done")
	p.Stop()
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
