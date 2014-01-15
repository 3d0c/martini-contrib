package linker

// Named connections manager.
// See the linker_test.json for available config options.

import (
	"github.com/3d0c/martini-contrib/config"
	"labix.org/v2/mgo"
	"log"
	// "runtime/debug"
)

type Link struct {
	Spec    string      `json:"spec"`
	DbName  string      `json:"db_name"`
	Default bool        `json:"default"`
	session interface{} `json:"-"`
}

type Links map[string]Link

type Config struct {
	Links `json:"connections"`
}

var pool *Config

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
}

func Get(args ...string) Link {
	var name string

	if pool == nil {
		pool = &Config{}

		config.LoadInto(pool)

		if len(pool.Links) == 0 {
			log.Println("Warning! 'connections' is not configured. Check, is there a 'connections' entry in the config file.")
		}
	}

	if len(args) > 0 {
		name = args[0]
	} else {
		name = getDefaultLink(pool.Links)
	}

	link, ok := pool.Links[name]
	if !ok {
		log.Printf("Connection with name '%s' not found.", name)
		return Link{}
	}

	if link.session == nil {
		var err error

		link.session, err = mgo.Dial(link.Spec)
		if err != nil {
			log.Println("Connection error:", err)
			return link
		}
	}

	return link
}

func (this Link) Session() interface{} {
	return this.session
}

func (this Link) MongoSession() *mgo.Session {
	return this.session.(*mgo.Session)
}

func (this Link) MongoDB(args ...string) *mgo.Database {
	name := this.DbName

	if len(args) > 0 {
		name = args[0]
	}

	return this.MongoSession().DB(name)
}

func getDefaultLink(l Links) string {
	for name, link := range l {
		if link.Default {
			return name
		}
	}

	panic("No default connection found.")
}
