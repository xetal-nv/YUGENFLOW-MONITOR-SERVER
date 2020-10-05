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

//var keysCol, dataCol, modelCol, latestCol, refreshCol, geoExternalsDB, geoDB, correctionDB *mongo.Collection
var spaceDB, shadowDataDB, entryDB, gateDB, stateDB *mongo.Collection

const (
	TO = 10
	DB = "yugenflow"
)

func Start() (err error) {
	if globals.DBSLogger, err = mlogger.DeclareLog("yfserver_DBS", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yfserver_DBS logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.DBSLogger, 80, 20, 10); e != nil {
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
		ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
		if err = client.Connect(ctx); err != nil {
			return
		}
		ctx, _ = context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
		if err = client.Ping(ctx, nil); err != nil {
			return
		}
		// Create/load collections
		spaceDB = client.Database(DB).Collection("spaceDB")
		shadowDataDB = client.Database(DB).Collection("shadowDataDB")
		entryDB = client.Database(DB).Collection("entryDB")
		gateDB = client.Database(DB).Collection("gateDB")
		stateDB = client.Database(DB).Collection("stateDB")
	}
	mlogger.Info(globals.DBSLogger,
		mlogger.LoggerData{"coreDBS.Start", "service started",
			[]int{0}, true})
	return
}

func Disconnect() error {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	mlogger.Info(globals.DBSLogger,
		mlogger.LoggerData{"coreDBS.Start", "service stopped",
			[]int{0}, true})
	return client.Disconnect(ctx)
}
