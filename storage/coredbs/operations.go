package coredbs

import (
	"context"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

func SaveSpaceData(nd dataformats.SpaceState) error {
	if globals.DisableDatabase {
		return nil
	}
	//fmt.Printf("TBD: Store space data %+v\n\n", nd)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	if _, err := dataDB.InsertOne(ctx, nd); err != nil {
		return err
	} else {
		return nil
	}
}

func SaveReferenceData(nd dataformats.SimpleSample) error {
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

// TODO move to cache !!!!
func SaveSpaceState(nd dataformats.SpaceState) (err error) {
	//fmt.Printf("TBD: Store space %v state %+v\n", entryName, nd)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	filter := bson.M{"id": bson.M{"$eq": nd.Id}}
	_, err = stateDB.DeleteMany(ctx, filter)
	if err != nil {
		return
	}
	ctx, _ = context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	if _, err := stateDB.InsertOne(ctx, nd); err != nil {
		return err
	} else {
		return nil
	}
}

// TODO move to cache !!!!
func LoadSpaceState(spaceName string) (state dataformats.SpaceState, err error) {
	//fmt.Printf("TBD: Load space %v state\n", spaceName)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	err = stateDB.FindOne(ctx, bson.M{"id": spaceName}).Decode(&state)
	return
}

// TODO move to cache !!!!
func SaveSpaceShadowState(nd dataformats.SpaceState) (err error) {
	//fmt.Printf("TBD: Store space %v state %+v\n", nd.Id, nd)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	filter := bson.M{"id": bson.M{"$eq": nd.Id}}
	_, err = shadowStateDB.DeleteMany(ctx, filter)
	if err != nil {
		return
	}
	ctx, _ = context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	if _, err := shadowStateDB.InsertOne(ctx, nd); err != nil {
		return err
	} else {
		return nil
	}
}

// TODO move to cache !!!!
func LoadSpaceShadowState(spaceName string) (state dataformats.SpaceState, err error) {
	//fmt.Printf("TBD: Load space %v state\n", entryName)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(TO)*time.Second)
	err = stateDB.FindOne(ctx, bson.M{"id": spaceName}).Decode(&state)
	return
}
