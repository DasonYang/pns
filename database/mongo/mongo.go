package mongo

import (
	"fmt"
	"log"
	"time"
	// "kpns/database"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// func makeQuery(query map[string]interface{}) bson.M {

//     var queries = make(bson.M)

//     for key := range query {
//         fmt.Printf("Function : makeQuery, key = %v\n", key)

//         if value, ok := query[key].(map[string]interface{}); ok {
//             // fmt.Println("Function : makeQuery, value is dict")
//             queries[key] = makeQuery(value)
//         } else if value, ok := query[key].([]map[string]interface{}); ok {
//             // fmt.Println("Function : makeQuery, value is dict array")
//             var q []bson.M
//             for _, v := range value { q = append(q, makeQuery(v)) }
//             queries[key] = q
//         } else {
//             // fmt.Println("Function : makeQuery, value is interface : ", query[key])
//             queries[key] = query[key]
//         }
//     }

//     return queries
// }

// func makeQuery(query map[string]interface{}) bson.M {

//     var queries = make(bson.M)

//     for key := range query {
//         fmt.Printf("Function : makeQuery, key = %v\n", key)

//         switch key{
//         case "$or":
//             if value, ok := query[key].(map[string]interface{}); ok {
//                 var l []bson.M
//                 for kk := range value {
//                     l = append(l, bson.M{kk:value[kk]})
//                 }
//                 queries[key] = l
//             }
//         default:
//             if value, ok := query[key].(string); ok {
//                 queries[key] = value
//             } else if value, ok := query[key].(map[string]interface{}); ok {
//                 queries[key] = makeQuery(value)
//                 // for v := range value {
//                 //     switch v {
//                 //     case "$regex":
//                 //         queries[key] = bson.RegEx{value[v].(string), "i"}
//                 //     case "$exists":
//                 //         queries[key] = bson.M{"$exists" : value[v].(bool)}
//                 //     }
//                 // }

//             }
//         }
//     }

//     fmt.Printf("Function : makeQuery, queries = %v\n", queries)

//     return queries
// }

func New() *DatabaseClient {
	session, err := mgo.Dial("localhost")

	session.SetSocketTimeout(1 * time.Hour)

	if err != nil {
		log.Fatal(err)
	}

	session.SetMode(mgo.Monotonic, true)

	return &DatabaseClient{Session: session}
}

type DatabaseClient struct {
	Session *mgo.Session
}

func (client *DatabaseClient) ReadOne(db string, collection string, query map[string]interface{}) map[string]interface{} {
	var result map[string]interface{}

	c := client.Session.DB(db).C(collection)

	err := c.Find(query).One(&result)
	if err != nil {
		log.Println(err)
	}

	return result
}

// ReadAll - Read data from collection
func (client *DatabaseClient) ReadAll(db string, collection string, query map[string]interface{}, condition map[string]interface{}) ([]map[string]interface{}, int) {
	var result []map[string]interface{}

	// fmt.Printf("query = %v, condiction = %v\n", query, condition)

	c := client.Session.DB(db).C(collection)
	q := c.Find(query)

	count, _ := q.Count()

	if s, ok := condition["sort"]; ok {
		// key or -key
		sort := fmt.Sprintf("%v", s)
		q = q.Sort(sort)
	}

	if s, ok := condition["skip"]; ok {
		q = q.Skip(s.(int))
	}

	if l, ok := condition["limit"]; ok {
		q = q.Limit(l.(int))
	}

	err := q.All(&result)
	if err != nil {
		log.Println(err)
	}

	return result, count
}

// Write - Write data to collection
func (client *DatabaseClient) Write(db string, collection string, data map[string]interface{}, params map[string]interface{}) error {
	log.Printf("Insert data into mongo : db = %v, collection = %v\n", db, collection)
	c := client.Session.DB(db).C(collection)
	key := fmt.Sprintf("%v", data["key"])

	if param, ok := params["ensureIndex"]; ok {
		// Index
		index := mgo.Index{
			Key:        []string{"key"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     true,
		}

		if dict, f := param.(map[string]interface{}); f {
			if keys, g := dict["keys"]; g {
				index.Key = keys.([]string)
			}
			if unique, g := dict["unique"]; g { // Unique
				index.Unique = unique.(bool)
			}
			if dropdups, g := dict["dropdups"]; g { //DropDups
				index.Unique = dropdups.(bool)
			}
			if bg, g := dict["bg"]; g { // Background
				index.Background = bg.(bool)
			}
			if sparse, g := dict["sparse"]; g {
				index.Sparse = sparse.(bool)
			}
			if expir, g := dict["expir"]; g {
				index.ExpireAfter = expir.(time.Duration)
			}
		}
		err := c.EnsureIndex(index)
		if err != nil {
			panic(err)
		}
	} else {
		err := c.EnsureIndexKey("key")
		if err != nil {
			panic(err)
		}
	}

	/*
	   type ChangeInfo struct {
	       Updated    int         // Number of existing documents updated
	       Removed    int         // Number of documents removed
	       UpsertedId interface{} // Upserted _id field, when not explicitly provided
	   }
	*/

	info, err := c.Upsert(bson.M{"key": key}, data)

	log.Printf("key = %v, updated = %v, removed = %v, upsertedId = %v\n", key, info.Updated, info.Removed, info.UpsertedId)

	return err
}

// Count - Get queryset count
func (client *DatabaseClient) Count(db string, collection string, query map[string]interface{}) int {
	c := client.Session.DB(db).C(collection)
	count, err := c.Find(query).Count()
	if err != nil {
		log.Println(err)
	}

	return count
}

// Update - Update or insert content to collection
func (client *DatabaseClient) Update(db string, collection string, query map[string]interface{}, content map[string]interface{}, params map[string]interface{}) error {
	c := client.Session.DB(db).C(collection)
	fmt.Printf("query = %v, content = %v\n", query, content)

	if param, ok := params["ensureIndex"]; ok {
		// Index
		index := mgo.Index{
			Key:        []string{"key"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     true,
		}

		if dict, f := param.(map[string]interface{}); f {
			if keys, g := dict["keys"]; g {
				index.Key = keys.([]string)
			}
			if unique, g := dict["unique"]; g { // Unique
				index.Unique = unique.(bool)
			}
			if dropdups, g := dict["dropdups"]; g { //DropDups
				index.Unique = dropdups.(bool)
			}
			if bg, g := dict["bg"]; g { // Background
				index.Background = bg.(bool)
			}
			if sparse, g := dict["sparse"]; g {
				index.Sparse = sparse.(bool)
			}
			if expir, g := dict["expir"]; g {
				index.ExpireAfter = expir.(time.Duration)
			}
		}
		err := c.EnsureIndex(index)
		if err != nil {
			panic(err)
		}
	} else {
		err := c.EnsureIndexKey("key")
		if err != nil {
			panic(err)
		}
	}

	info, err := c.Upsert(query, content)
	// err := c.Update(bson.M{"key":"enUS"}, bson.M{"$unset":bson.M{"100":""}})

	// err := c.Update(query, content, params)

	log.Printf("updated = %v, removed = %v, upsertedId = %v\n", info.Updated, info.Removed, info.UpsertedId)

	if err != nil {
		log.Printf("error = %v\n", err)
	}

	return err
}

// Delete - Delete record in collection
func (client *DatabaseClient) Delete(db string, collection string, query map[string]interface{}) error {

	log.Printf("Delete : query = %v\n", query)
	c := client.Session.DB(db).C(collection)

	err := c.Remove(query)

	if err != nil {
		log.Printf("error = %v\n", err)
	}

	return err
}

// BulkWrite - Write bulk data into collection
func (client *DatabaseClient) BulkWrite(db string, collection string, data []interface{}) error {

	c := client.Session.DB(db).C(collection)
	b := c.Bulk()

	// Index
	// index := mgo.Index{
	//     Key:        []string{"key"},
	//     Unique:     true,
	//     DropDups:   true,
	//     Background: true,
	//     Sparse:     true,
	// }

	{ // Ensure index
		err := c.EnsureIndexKey("key")
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("data = %v\n", data...)

	b.Insert(data...)

	res, err := b.Run()

	fmt.Printf("res = %v\n", res)

	return err
}
