package main

import (
	"errors"
	"fmt"
	"learning-web-chatboard4/common"
	"learning-web-chatboard4/common/models"
	"time"
)

const (
	topicsTable      = "topics"
	repliesTable     = "replies"
	descendingUpdate = "last_update"
)

func createTopic(topic *models.Topic, corrId string) {
	err := createTopicInternal(topic)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, topic, "Topic", corrId)
}

func createTopicInternal(topic *models.Topic) (err error) {
	if common.IsEmpty(topic.Topic, topic.Owner) {
		err = errors.New("contains empty string")
		return
	}
	now := time.Now()
	topic.UuId = common.NewUuIdString()
	topic.LastUpdate = now
	topic.CreatedAt = now
	err = createTopicSQL(topic)
	return
}

func createTopicSQL(topic *models.Topic) (err error) {
	affected, err := dbEngine.
		Table(topicsTable).
		InsertOne(&topic)
	if err == nil && affected != 1 {
		err = fmt.Errorf(
			"something wrong. returned value was %d",
			affected,
		)
	}
	return
}

func createReply(reply *models.Reply, corrId string) {
	err := createReplyInternal(reply)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, reply, "Reply", corrId)
}

func createReplyInternal(reply *models.Reply) (err error) {
	if common.IsEmpty(
		reply.Body,
		reply.Contributor,
	) {
		err = errors.New("contains empty string")
		return
	}
	reply.UuId = common.NewUuIdString()
	reply.CreatedAt = time.Now()
	err = createReplySQL(reply)
	return
}

func createReplySQL(reply *models.Reply) (err error) {
	affected, err := dbEngine.
		Table(repliesTable).
		InsertOne(reply)
	if err == nil && affected != 1 {
		err = fmt.Errorf(
			"something wrong. returned value was %d",
			affected,
		)
	}
	return
}

func readATopic(topic *models.Topic, corrId string) {
	err := readATopicInternal(topic)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, topic, "Topic", corrId)
}

func readATopicInternal(topic *models.Topic) (err error) {
	if common.IsEmpty(topic.UuId) {
		err = errors.New("need uuid for finding thread")
		return
	}
	err = readATopicSQL(topic)
	return
}

func readATopicSQL(topic *models.Topic) (err error) {
	ok, err := dbEngine.
		Table(topicsTable).
		Get(topic)
	if err == nil && !ok {
		err = errors.New("no such thread")
	}
	return
}

func updateTopic(topic *models.Topic, corrId string) {
	err := updateTopicInternal(topic)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, topic, "Topic", corrId)
}

func updateTopicInternal(topic *models.Topic) (err error) {
	if common.IsEmpty(
		topic.UuId,
		topic.Topic,
		topic.Owner,
	) {
		err = errors.New("contains empty string")
		return
	}
	topic.LastUpdate = time.Now()
	err = updateTopicSQL(topic)
	return
}

func updateTopicSQL(topic *models.Topic) (err error) {
	affected, err := dbEngine.
		Table(topicsTable).
		ID(topic.Id).
		Update(topic)
	if err == nil && affected != 1 {
		err = fmt.Errorf(
			"something wrong. returned value was %d",
			affected,
		)
	}
	return
}

func incrementTopic(topic *models.Topic, corrId string) {
	err := incrementTopicInternal(topic)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, topic, "Topic", corrId)
}

func incrementTopicInternal(topic *models.Topic) (err error) {
	err = readATopicInternal(topic)
	if err != nil {
		return
	}

	topic.NumReplies++
	err = updateTopicInternal(topic)
	return
}

func readRepliesInTopic(topic *models.Topic, corrId string) {
	// is there a way to check valid id before?
	replies, err := readRepliesInTopicSQL(topic)
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	}

	common.SendOK(server, &replies, "ReplySlice", corrId)
}

func readRepliesInTopicSQL(topic *models.Topic) (posts []models.Reply, err error) {
	err = dbEngine.
		Table(repliesTable).
		Where("topic_id = ?", topic.Id).
		Find(&posts)
	return
}

func readTopics(corrId string) {
	topics, err := readTopicsSQL()
	if err != nil {
		common.HandleError(server, logger, err.Error(), corrId)
		return
	} else {
		common.SendOK(server, &topics, "TopicSlice", corrId)
	}
}

func readTopicsSQL() (topics []models.Topic, err error) {
	err = dbEngine.
		Table(topicsTable).
		Desc(descendingUpdate).
		Find(&topics)
	return
}
