// +build !embedded

package coredbs

import (
	"context"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func SaveSpaceData(nd dataformats.SpaceState) error {
	if globals.DisableDatabase {
		return nil
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	if _, err := dataDB.InsertOne(ctx, nd); err != nil {
		return err
	} else {
		return nil
	}
}

func SaveReferenceData(nd dataformats.MeasurementSample) error {
	if globals.DisableDatabase {
		return nil
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	if _, err := referenceDB.InsertOne(ctx, nd); err != nil {
		return err
	} else {
		return nil
	}
}

func SaveShadowSpaceData(nd dataformats.SpaceState) error {
	if globals.DisableDatabase {
		return nil
	}
	//fmt.Printf("TBD: Store shadow space data %+v\n", nd)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	if _, err := shadowDataDB.InsertOne(ctx, nd); err != nil {
		return err
	} else {
		return nil
	}
}

func ReadSpaceData(spacename string, howMany int) (result []dataformats.MeasurementSample, err error) {
	if howMany == 0 {
		return
	}
	filter := bson.D{{"id", spacename}}
	opt := options.Find()
	opt.SetSort(bson.D{{"ts", -1}})
	opt.SetLimit(int64(howMany))
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	cursor, er := dataDB.Find(ctx, filter, opt)
	if er != nil {
		err = er
		return
	}
	err = cursor.All(context.TODO(), &result)
	return
}

func ReadReferenceData(spacename string, howMany int) (result []dataformats.MeasurementSample, err error) {
	if howMany == 0 {
		return
	}
	filter := bson.D{{"space", spacename}}
	opt := options.Find()
	opt.SetSort(bson.D{{"ts", -1}})
	opt.SetLimit(int64(howMany))
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	cursor, er := referenceDB.Find(ctx, filter, opt)
	if er != nil {
		err = er
		return
	}
	err = cursor.All(context.TODO(), &result)
	return
}

func ReadReferenceDataSeries(spacename string, ts0, ts1 int) (result []dataformats.MeasurementSample, err error) {
	if ts1 <= ts0 {
		return
	}
	filter := bson.D{{"space", spacename},
		{"ts", bson.D{{"$lt", ts1}, {"$gt", ts0}}},
	}
	opt := options.Find()
	opt.SetSort(bson.D{{"ts", -1}})
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	cursor, er := referenceDB.Find(ctx, filter, opt)
	if er != nil {
		err = er
		return
	}
	err = cursor.All(context.TODO(), &result)
	return
}

func ReadSpaceDataSeries(spacename string, ts0, ts1 int) (result []dataformats.MeasurementSample, err error) {
	if ts1 <= ts0 {
		return
	}
	filter := bson.D{{"id", spacename},
		{"ts", bson.D{{"$lt", ts1}, {"$gt", ts0}}},
	}
	opt := options.Find()
	opt.SetSort(bson.D{{"ts", -1}})
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	cursor, er := dataDB.Find(ctx, filter, opt)
	if er != nil {
		err = er
		return
	}
	err = cursor.All(context.TODO(), &result)
	return
}

func VerifyPresence(spacename string, ts0, ts1 int) (present bool, err error) {
	if ts1 <= ts0 {
		return
	}
	filter := bson.D{{"id", spacename},
		{"count", bson.D{{"$ne", 0}}},
		{"ts", bson.D{{"$lt", ts1}, {"$gt", ts0}}},
	}
	opt := options.Find()
	opt.SetSort(bson.D{{"ts", -1}})
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	count, er := dataDB.CountDocuments(ctx, filter)
	if er != nil {
		err = er
		return
	}
	present = count >= 2
	return
}
