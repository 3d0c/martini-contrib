package model

// Simple model layer. It implements base Create, Find, Update, Delete methods.
//
// @TODO: Only abstract methods should be here. Implement all backend queries
//        inside specific adapters in the linker package.

import (
	"fmt"
	"github.com/3d0c/martini-contrib/linker"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"reflect"
	"strings"
)

// 'collection'
// 		Collection name. By default it's made from SchemeName. E.g. 'UserScheme'
//		becames 'Users' collection, simply by trimming 'Scheme' literal and adding
//		's' to tne end.
//
// Arguments list is optional:
// 		It expects a map[string]interface{}, containing following members (ony one so far):
// 		'collection'	(string) - collection name
//
// E.g.: New(UserScheme{}, map[string]interface{"collection": "accounts"})
//
type Model struct {
	scheme     interface{}
	collection string
}

type Query struct {
	scheme interface{}
	*mgo.Query
}

var qCache map[string]*Query

type Iter struct {
	*mgo.Iter
}

func New(scheme interface{}, args ...interface{}) *Model {
	var config map[string]interface{}

	if len(args) > 0 {
		config = args[0].([]interface{})[0].(map[string]interface{})
	}

	this := &Model{
		scheme:     scheme,
		collection: collectionName(scheme),
	}

	// Overriding collection name, if there is a 'collection' key in config map
	if name, ok := config["collection"]; ok {
		this.collection = name.(string)
	}

	qCache = make(map[string]*Query, 0)

	return this
}

// empty for all or id (as a bson.ObjectId or string)
// all other arguments are the same as mgo Find method applies (map, struct)
func (this *Model) Find(args ...interface{}) *Query {
	var query interface{}

	if len(args) == 0 {
		query = bson.M{}
	} else {
		query = prepareQuery(args[0])
	}

	key := fmt.Sprintf("%v", query)

	q, found := qCache[key]
	if !found {
		qCache[key] = &Query{scheme: this.scheme}
		qCache[key].Query = linker.Get().MongoDB().C(this.collection).Find(query)

		return qCache[key]
	}

	return q
}

func (this *Model) Expand(result interface{}, fieldTag string) {
	resultv := reflect.ValueOf(result)

	if resultv.Kind() == reflect.Ptr && resultv.Elem().Kind() == reflect.Struct {
		this.expandField(getFieldByTag(reflect.TypeOf(result).Elem(), resultv, fieldTag))
		return
	}

	if resultv.Kind() == reflect.Slice {
		for i := 0; i < resultv.Len(); i++ {
			this.expandField(getFieldByTag(reflect.TypeOf(result).Elem(), resultv.Index(i), fieldTag))
		}
		return
	}

	log.Println("Unable to expand. Unexpected type:", resultv.Kind())
	return
}

func (this *Model) expandField(field reflect.Value, typ reflect.StructField) {
	if !field.IsValid() {
		log.Println("Unable to expand. Field for tag", field.Type().Name(), "not found.")
		return
	}

	if !field.CanSet() {
		log.Println("Unable to expand. Can't set field:", field.Type().Name())
		return
	}

	if field.Kind() != reflect.Interface {
		log.Println("Unable to expand. Field should be an interface,", field.Kind(), "given.")
		return
	}

	r := this.Find(field.Elem().Interface().(bson.ObjectId)).One()

	field.Set(reflect.ValueOf(r))
}

func getFieldByTag(typ reflect.Type, val reflect.Value, tag string) (reflect.Value, reflect.StructField) {
	for i := 0; i < typ.NumField(); i++ {
		if jsonTags := strings.Split(typ.Field(i).Tag.Get("json"), ","); jsonTags[0] == tag {
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			return val.Field(i), typ.Field(i)
		}
	}

	return reflect.Value{}, reflect.StructField{}
}

func (q *Query) Skip(n int) *Query {
	q.Query = q.Query.Skip(n)
	return q
}

func (q *Query) Limit(n int) *Query {
	if n == 0 {
		n = 1000
	}

	q.Query = q.Query.Limit(n)
	return q
}

func (q *Query) All() interface{} {
	result := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(q.scheme)), 0, 0).Interface()

	iter := &Iter{Iter: q.Iter()}
	err := iter.All(&result)
	if err != nil {
		log.Println("Query error:", err)
		return nil
	}

	return result
}

// This is an overridden version of All() iterator, because original one doesn't support dynamically created
// slices. When it does, this stuff could be removed.
func (iter *Iter) All(result interface{}) error {
	resultv := reflect.ValueOf(result)
	if resultv.Kind() != reflect.Ptr || (resultv.Elem().Kind() != reflect.Slice && resultv.Elem().Kind() != reflect.Interface) {
		panic("result argument must be a slice address")
	}
	slicev := resultv.Elem()

	if resultv.Elem().Kind() == reflect.Interface {
		slicev = slicev.Elem().Slice(0, slicev.Elem().Cap())
	} else {
		slicev = slicev.Slice(0, slicev.Elem().Cap())
	}

	elemt := slicev.Type().Elem()
	i := 0
	for {
		if slicev.Len() == i {
			elemp := reflect.New(elemt)
			if !iter.Next(elemp.Interface()) {
				break
			}
			slicev = reflect.Append(slicev, elemp.Elem())
			slicev = slicev.Slice(0, slicev.Cap())
		} else {
			if !iter.Next(slicev.Index(i).Addr().Interface()) {
				break
			}
		}
		i++
	}
	resultv.Elem().Set(slicev.Slice(0, i))
	return iter.Close()
}

func (q *Query) One() interface{} {
	result := reflect.New(reflect.TypeOf(q.scheme))

	err := q.Query.One(result.Interface())
	if err != nil {
		log.Println("Query error:", err)
		return nil
	}

	return result.Interface()
}

func prepareQuery(i interface{}) interface{} {
	switch i.(type) {
	case string:
		if bson.IsObjectIdHex(i.(string)) {
			return bson.M{"_id": bson.ObjectIdHex(i.(string))}
		}
		break

	case bson.ObjectId:
		return bson.M{"_id": i.(bson.ObjectId)}

	case []bson.ObjectId:
		return bson.M{"_id": bson.M{"$in": i.([]bson.ObjectId)}}

	default:
		return i
	}

	return i
}

func (this *Model) Create(query interface{}) interface{} {
	if query == nil {
		log.Println("query is nil")
		return nil
	}

	// We should generate and inject '_id' field into the query.
	id := bson.NewObjectId()
	queryKind := reflect.TypeOf(query).Kind()

	switch queryKind {
	case reflect.Map:
		query.(map[string]interface{})["_id"] = id
		break

	case reflect.Ptr:
		queryValue := reflect.ValueOf(query).Elem()
		schemeType := reflect.TypeOf(this.scheme)
		if _, ok := schemeType.FieldByName("Id"); !ok {
			log.Println("Scheme", schemeType.Name(), "doesn't have a required Id field.")
			return nil
		}

		field := queryValue.FieldByName("Id")
		if !field.CanSet() {
			log.Println("Unable set value to Id field of ", schemeType.Name())
			return nil
		}

		field.Set(reflect.ValueOf(id))

		break

	default:
		log.Println("Unknown kind of query:", queryKind)
		return nil
	}

	err := linker.Get().MongoDB().C(this.collection).Insert(query)
	if err != nil {
		log.Println(err)
		return nil
	}

	return this.Find(id).One()
}

func (this *Model) Update(selector interface{}, query interface{}) (interface{}, error) {
	tmp := bson.M{}

	if _, err := linker.Get().MongoDB().C(this.collection).Find(prepareQuery(selector)).Apply(
		mgo.Change{Update: bson.M{"$set": query}},
		&tmp,
	); err != nil {
		log.Println("Unable to update:", selector, "with:", query, ", error:", err)
		return nil, err
	}

	result := reflect.New(reflect.TypeOf(this.scheme))

	if err := linker.Get().MongoDB().C(this.collection).FindId(tmp["_id"]).One(result.Interface()); err != nil {
		log.Println("Unable to find updated:", selector, "error:", err)
		return nil, err
	}

	return result.Interface(), nil
}

func (this *Model) Delete(selector interface{}) bool {
	if _, err := linker.Get().MongoDB().C(this.collection).RemoveAll(selector); err != nil {
		log.Println("Unable to delete selector:", selector, ", error:", err)
		return false
	}

	return true
}

func collectionName(i interface{}) string {
	s := reflect.TypeOf(i).Name()

	return strings.Title(strings.TrimRight(s, "Scheme")) + "s"
}
