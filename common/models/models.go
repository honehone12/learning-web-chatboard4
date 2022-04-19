package models

import (
	"encoding/base64"
	"time"
)

type User struct {
	Id        uint      `xorm:"pk autoincr 'id'" json:"id"`
	UuId      string    `xorm:"not null unique 'uu_id'" json:"uuid"`
	Name      string    `xorm:"not null unique 'name'" json:"name"`
	Email     string    `xorm:"not null unique 'email'" json:"email"`
	Password  string    `xorm:"not null 'password'" json:"password"`
	Salt      string    `xorm:"not null 'salt'" json:"salt"`
	Token     string    `xorm:"TEXT 'token'" json:"token"`
	CreatedAt time.Time `xorm:"not null 'created_at'" json:"created_at"`
}

type Session struct {
	UuId       string    `xorm:"not null unique 'uu_id'" json:"uuid"`
	State      string    `xorm:"TEXT 'state'" json:"state"`
	TopicId    uint      `xorm:"topic_id" json:"topic_id"`
	TopicUuId  string    `xorm:"topic_uu_id" json:"topic_uuid"`
	Token      string    `xorm:"TEXT 'token'" json:"token"`
	UserName   string    `xorm:"user_name" json:"user_name"`
	UserEmail  string    `xorm:"user_email" json:"user_email"`
	UserId     uint      `xorm:"user_id" json:"user_id"`
	NumErrors  uint      `xorm:"num_errors" json:"num_errors"`
	LastUpdate time.Time `xorm:"not null 'last_update'" json:"last_update"`
	CreatedAt  time.Time `xorm:"not null 'created_at'" json:"created_at"`
}

type Topic struct {
	Id         uint      `xorm:"pk autoincr 'id'" json:"id"`
	UuId       string    `xorm:"not null unique 'uu_id'" json:"uuid"`
	Topic      string    `xorm:"TEXT 'topic'" json:"topic"`
	NumReplies uint      `xorm:"num_replies" json:"num_replies"`
	Owner      string    `xorm:"owner" json:"owner"`
	UserId     uint      `xorm:"user_id" json:"user_id"`
	LastUpdate time.Time `xorm:"not null 'last_update'" json:"last_update"`
	CreatedAt  time.Time `xorm:"not null 'created_at'" json:"created_at"`
}

type Reply struct {
	Id          uint      `xorm:"pk autoincr 'id'" json:"id"`
	UuId        string    `xorm:"not null unique 'uu_id'" json:"uuid"`
	Body        string    `xorm:"TEXT 'body'" json:"body"`
	Contributor string    `xorm:"contributor" json:"contributor"`
	UserId      uint      `xorm:"user_id" json:"user_id"`
	TopicId     uint      `xorm:"topic_id" json:"topic_id"`
	CreatedAt   time.Time `xorm:"not null 'created_at'" json:"created_at"`
}

func (topic *Topic) When() string {
	return topic.CreatedAt.Format("2006/Jan/2 at 3:04pm")
}

func (reply *Reply) When() string {
	return reply.CreatedAt.Format("2006/Jan/2 at 3:04pm")
}

func (topic *Topic) AsURL() string {
	return base64.URLEncoding.EncodeToString([]byte(topic.UuId))
}
