// +build !embedded

package coredbs

import (
	"context"
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

var client *mongo.Client

var dataDB, referenceDB, shadowDataDB *mongo.Collection

const (
	TO = 10
	DB = "yugenflow"
)

func Start() (err error) {
	if globals.DisableDatabase {
		return
	}

	if globals.DBSLog, err = mlogger.DeclareLog("yugenflow_DBS", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_DBS logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.DBSLog, 80, 20, 10); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	var clientOptions *options.ClientOptions
	if globals.DBUser != "" {
		clientOptions = options.Client().ApplyURI(globals.DBpath).
			SetAuth(options.Credential{
				AuthSource: DB, Username: globals.DBUser, Password: globals.DBUserPassword,
			})
	} else {
		clientOptions = options.Client().ApplyURI(globals.DBpath)
	}
	if client, err = mongo.NewClient(clientOptions); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
		defer cancel()
		if err = client.Connect(ctx); err != nil {
			return
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
		defer cancel2()

		if err = client.Ping(ctx2, nil); err != nil {
			return
		}
		// Create/load collections
		dataDB = client.Database(DB).Collection("dataDB")
		referenceDB = client.Database(DB).Collection("referenceDB")
		shadowDataDB = client.Database(DB).Collection("shadowDataDB")
	}
	mlogger.Info(globals.DBSLog,
		mlogger.LoggerData{"coreDBS.Start", "service started",
			[]int{0}, true})
	return
}

func Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	defer cancel()
	mlogger.Info(globals.DBSLog,
		mlogger.LoggerData{"coreDBS.Start", "service stopped",
			[]int{0}, true})
	return client.Disconnect(ctx)
}
