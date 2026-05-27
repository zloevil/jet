package mongodb

import "go.mongodb.org/mongo-driver/v2/bson"

func BsonEncode(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func BsonDecode[T any](payload []byte) (*T, error) {
	var res T
	err := bson.Unmarshal(payload, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
