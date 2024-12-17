package kafkax

import (
	"encoding/json"

	"github.com/niudaii/goutil/reflectx"
)

type HandlerBinding struct {
	TopicName   string
	HandlerName string
	HandlerFunc HandlerFunc
	Handler     string
}

func DirectBinding(topicName, handlerName string, handlerFunc HandlerFunc) HandlerBinding {
	return HandlerBinding{
		TopicName:   topicName,
		HandlerName: handlerName,
		HandlerFunc: handlerFunc,
		Handler:     reflectx.GetFuncName(handlerFunc),
	}
}

func BindJSON(msg []byte, obj any) (err error) {
	err = json.Unmarshal(msg, obj)
	return
}
